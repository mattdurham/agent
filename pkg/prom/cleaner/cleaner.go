package cleaner

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/grafana/agent/pkg/prom/instance"
	"github.com/grafana/agent/pkg/prom/wal"
	promwal "github.com/prometheus/prometheus/tsdb/wal"
)

type walCleaner struct {
	logger          log.Logger
	walDirectory    string
	instanceManager instance.Manager
	minAge          time.Duration
	ticker          *time.Ticker
	done            chan bool
}

type Cleaner interface {
	// Delete any storage (Prometheus WAL) no longer associated with a instance.ManagedInstance
	CleanupStorage() error

	// Stop any tasks running
	Stop()
}

// Create a new Cleaner implementation that looks for abandoned WALs in the given
// directory and removes them if they haven't been modified in over minAge. Starts
// a goroutine to periodically run Cleaner.CleanupStorage in a loop
func NewCleaner(logger log.Logger, manager instance.Manager, walDirectory string, minAge time.Duration, period time.Duration) Cleaner {
	c := &walCleaner{
		logger:          logger,
		instanceManager: manager,
		walDirectory:    walDirectory,
		minAge:          minAge,
		ticker:          time.NewTicker(period),
		done:            make(chan bool),
	}

	go c.run()
	return c
}

// Get storage directories used for each ManagedInstance
func (c *walCleaner) getManagedStorage(instances map[string]instance.ManagedInstance) map[string]bool {
	out := make(map[string]bool)

	for _, inst := range instances {
		out[inst.StorageDirectory()] = true
	}

	return out
}

// Get all Storage directories under walDirectory
func (c *walCleaner) getAllStorage() []string {
	var out []string
	root := c.walDirectory

	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			// Just log any errors traversing the WAL directory. This will potentially result
			// in a WAL (that has incorrect permissions somehow) not being cleaned up. This is
			// better than preventing *all* other WALs from being cleaned up.
			level.Warn(c.logger).Log("msg", "unable to traverse WAL storage path", "path", p, "err", err)
		} else if info.IsDir() && path.Dir(p) == root {
			// Directories that are a single level below the root for all WALs
			out = append(out, p)
		}
		return nil
	})

	return out
}

// Get the mtime of the most recent WAL segment based on the Storage directory
func (c *walCleaner) lastWrittenTime(storage string) (time.Time, error) {
	walDir := wal.SubDirectory(storage)
	empty := time.Time{}

	existing, err := promwal.Open(c.logger, walDir)
	if err != nil {
		return empty, err
	}

	// We don't care if there are errors closing the abandoned WAL
	defer func() { _ = existing.Close() }()

	_, last, err := existing.Segments()
	if err != nil {
		return empty, err
	}

	if last == -1 {
		return empty, fmt.Errorf("unable to determine most recent segment for %s", walDir)
	}

	// full path to the most recent segment in this WAL
	lastSegment := promwal.SegmentName(walDir, last)
	segmentFile, err := os.Stat(lastSegment)
	if err != nil {
		return empty, err
	}

	return segmentFile.ModTime(), nil
}

// Get the full path of storage directories that aren't referenced by any instance.ManagedInstance
// and haven't been written to within a configured duration (usually several hours or more).
func (c *walCleaner) abandonedStorage(instances map[string]instance.ManagedInstance, now time.Time) []string {
	var out []string

	managed := c.getManagedStorage(instances)
	all := c.getAllStorage()

	for _, dir := range all {
		if !managed[dir] {
			mtime, err := c.lastWrittenTime(dir)
			if err != nil {
				level.Warn(c.logger).Log("msg", "unable to find segment mtime of WAL", "name", dir, "err", err)
				continue
			}

			diff := now.Sub(mtime)
			if diff > c.minAge {
				// The last segment for this WAL was modified more then $minAge (positive number of hours)
				// in the past. This makes it a candidate for deletion since it's also not associated with
				// any Instances this agent knows about.
				out = append(out, dir)
			}

			level.Debug(c.logger).Log("msg", "abandoned WAL", "name", dir, "mtime", mtime, "diff", diff)
		} else {
			level.Debug(c.logger).Log("msg", "active WAL", "name", dir)
		}
	}

	return out
}

func (c *walCleaner) run() {
	for {
		select {
		case <-c.done:
			return
		case <-c.ticker.C:
			err := c.CleanupStorage()
			if err != nil {
				level.Error(c.logger).Log("msg", "cleanup failed", "err", err)
			}
		}
	}
}

func (c *walCleaner) CleanupStorage() error {
	instances := c.instanceManager.ListInstances()
	abandoned := c.abandonedStorage(instances, time.Now())

	for _, a := range abandoned {
		// TODO(nickp) actually remove instead of logging
		level.Info(c.logger).Log("msg", "would delete WAL", "name", a)
	}

	return nil
}

func (c *walCleaner) Stop() {
	level.Debug(c.logger).Log("msg", "stopping cleaner...")
	c.ticker.Stop()
	close(c.done)
}
