package windows_exporter //nolint:golint

import (
	"reflect"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/prometheus-community/windows_exporter/collector"

	"github.com/go-kit/kit/log"
	"github.com/grafana/agent/pkg/integrations"
	"github.com/grafana/agent/pkg/integrations/config"
)

func init() {
	integrations.RegisterIntegration(&Config{})
}

// Config controls the windows_exporter integration.
// All of these and their child fields are pointers so we can determine if the value was set or not.
type Config struct {
	Common config.Common `yaml:",inline"`

	EnabledCollectors string `yaml:"enabled_collectors"`

	Exchange    *ExchangeConfig    `yaml:"exchange"`
	IIS         IISConfig          `yaml:"iis"`
	TextFile    *TextFileConfig    `yaml:"text_file"`
	SMTP        *SMTPConfig        `yaml:"smtp"`
	Service     *ServiceConfig     `yaml:"service"`
	Process     *ProcessConfig     `yaml:"process"`
	Network     *NetworkConfig     `yaml:"network"`
	MSSQL       *MSSQLConfig       `yaml:"mssql"`
	MSMQ        *MSMQConfig        `yaml:"msmq"`
	LogicalDisk *LogicalDiskConfig `yaml:"logical_disk"`
}

func (c *Config) Name() string {
	return "windows_exporter"
}

func (c *Config) CommonConfig() config.Common {
	return c.Common
}

func (c *Config) NewIntegration(l log.Logger) (integrations.Integration, error) {
	return New(l, c)
}

/*
// The Windows Collector takes a map of configuration to set, so we need to convert from agent config to a key value
// using the windows_exporter key name 'collector.iis.site-whitelist' for example.
func (c *Config) ConvertToMap() map[string]string {
	configMap := make(map[string]string)
	translateConfig(c.Exchange, configMap)
	translateConfig(c.IIS, configMap)
	translateConfig(c.LogicalDisk, configMap)
	translateConfig(c.MSMQ, configMap)
	translateConfig(c.MSSQL, configMap)
	translateConfig(c.Network, configMap)
	translateConfig(c.Process, configMap)
	translateConfig(c.Service, configMap)
	translateConfig(c.SMTP, configMap)
	translateConfig(c.TextFile, configMap)
	return configMap
}
*/
/*
func (c *Config) ApplyConfig(exporterConfigs map[string]collector.Config) {
	agentConfigs := []translatableConfig{
		c.Exchange,
		c.IIS,
		c.LogicalDisk,
		c.MSMQ,
		c.MSSQL,
		c.Network,
		c.Process,
		c.Service,
		c.SMTP,
		c.TextFile,
	}
	// Brute force the syncing
	for _, ac := range agentConfigs {
		if ac == nil || reflect.ValueOf(ac).IsNil() {
			continue
		}
		for _, ec := range exporterConfigs {
			// Sync will return true if it can handle the exporter config
			// which means we can break early
			if ac.Sync(ec) {
				break
			}
		}
	}
}*/

type ExchangeConfig struct {
	EnabledList *string `yaml:"enabled_list"`
}

func (c *ExchangeConfig) translate(cm map[string]string) {
	setIfNotNil(cm, "collectors.exchange.enabled", c.EnabledList)
}

func (c *ExchangeConfig) Sync(v interface{}) bool {
	other, ok := v.(*collector.ExchangeConfig)
	if ok {
		setStringIfNotNil(c.EnabledList, &other.Enabled)
	}
	return ok
}

func (c *Config) unmarshalYAML(unmarshal func(interface{}) error) error {
	type plain Config
	return unmarshal((*plain)(c))
}

func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {

	type plain Config
	return unmarshal((*plain)(c))
}

type IISConfig collector.IISConfig

func (c *IISConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	assignDefaults((*collector.IISConfig)(c))

	// Re-declare IISConfig with YAML tags. This can be converted to an IISConfig
	// freely as long as the types are structurally identical.
	type plain struct {
		SiteWhiteList string `yaml:"site_whitelist"`
		SiteBlackList string `yaml:"site_blacklist"`
		AppWhiteList  string `yaml:"app_whitelist"`
		AppBlackList  string `yaml:"app_blacklist"`
	}

	// Convert our existing IISConfig to a plain. This IISConfig should already
	// have default values from the kingpin via Config.UnmarshalYAML already
	// being called first. Note that YAML's unmarshaler doesn't fill in fields
	// that don't exist, so we don't need pointers here.
	p := plain(*c)
	if err := unmarshal(&p); err != nil {
		return err
	}
	*c = IISConfig(p)
	return nil
}

func (c *IISConfig) Sync(v interface{}) bool {
	/*other, ok := v.(*collector.IISConfig)
	if ok {
		*other = collector.IISConfig(*c)
	}
	return ok*/
	return false
}

func assignDefaults(c collector.Config) {
	// Create a fake application to force defaults to be applied
	app := kingpin.New("", "")
	c.RegisterKingpin(app)

	// Parse an empty set of flags, which will fill in default values for everything.
	_, err := app.Parse([]string{})
	if err != nil {
		panic(err)
	}
}

/*
type IISConfig struct {
	SiteWhiteList *string `yaml:"site_whitelist"`
	SiteBlackList *string `yaml:"site_blacklist"`
	AppWhiteList  *string `yaml:"app_whitelist"`
	AppBlackList  *string `yaml:"app_blacklist"`
}

func (c *IISConfig) translate(cm map[string]string) {
	setIfNotNil(cm, "collector.iis.site-whitelist", c.SiteWhiteList)
	setIfNotNil(cm, "collector.iis.site-blacklist", c.SiteBlackList)
	setIfNotNil(cm, "collector.iis.app-whitelist", c.AppWhiteList)
	setIfNotNil(cm, "collector.iis.app-blacklist", c.AppBlackList)
}

func (c *IISConfig) Sync(v interface{}) bool {
	other, ok := v.(*collector.IISConfig)
	if ok {
		setStringIfNotNil(c.SiteWhiteList, &other.SiteWhiteList)
		setStringIfNotNil(c.SiteBlackList, &other.SiteBlackList)
		setStringIfNotNil(c.AppWhiteList, &other.AppWhiteList)
		setStringIfNotNil(c.AppBlackList, &other.AppBlackList)
	}
	return ok
}*/

type TextFileConfig struct {
	TextFileDirectory *string `yaml:"text_file_directory"`
}

func (c *TextFileConfig) translate(cm map[string]string) {
	setIfNotNil(cm, "collector.textfile.directory", c.TextFileDirectory)
}

func (c *TextFileConfig) Sync(v interface{}) bool {
	other, ok := v.(*collector.TextFileConfig)
	if ok {
		setStringIfNotNil(c.TextFileDirectory, &other.TextFileDirectory)
	}
	return ok
}

type SMTPConfig struct {
	WhiteList *string `yaml:"whitelist"`
	BlackList *string `yaml:"blacklist"`
}

func (c *SMTPConfig) translate(cm map[string]string) {
	setIfNotNil(cm, "collector.smtp.server-whitelist", c.WhiteList)
	setIfNotNil(cm, "collector.smtp.server-blacklist", c.BlackList)
}

func (c *SMTPConfig) Sync(v interface{}) bool {
	other, ok := v.(*collector.SMTPConfig)
	if ok {
		setStringIfNotNil(c.WhiteList, &other.ServerWhiteList)
		setStringIfNotNil(c.BlackList, &other.ServerBlackList)
	}
	return ok
}

type ServiceConfig struct {
	Where *string `yaml:"where_clause"`
}

func (c *ServiceConfig) translate(cm map[string]string) {
	setIfNotNil(cm, "collector.service.services-where", c.Where)
}

func (c *ServiceConfig) Sync(v interface{}) bool {
	other, ok := v.(*collector.ServiceConfig)
	if ok {
		setStringIfNotNil(c.Where, &other.ServiceWhereClause)
	}
	return ok
}

type ProcessConfig struct {
	WhiteList *string `yaml:"whitelist"`
	BlackList *string `yaml:"blacklist"`
}

func (c *ProcessConfig) translate(cm map[string]string) {
	setIfNotNil(cm, "collector.process.whitelist", c.WhiteList)
	setIfNotNil(cm, "collector.process.blacklist", c.BlackList)
}

func (c *ProcessConfig) Sync(v interface{}) bool {
	other, ok := v.(*collector.ProcessConfig)
	if ok {
		setStringIfNotNil(c.WhiteList, &other.ProcessWhiteList)
		setStringIfNotNil(c.BlackList, &other.ProcessBlackList)
	}
	return ok
}

type NetworkConfig struct {
	WhiteList *string `yaml:"whitelist"`
	BlackList *string `yaml:"blacklist"`
}

func (c *NetworkConfig) translate(cm map[string]string) {
	setIfNotNil(cm, "collector.net.nic-whitelist", c.WhiteList)
	setIfNotNil(cm, "collector.net.nic-blacklist", c.BlackList)
}

func (c *NetworkConfig) Sync(v interface{}) bool {
	other, ok := v.(*collector.NetworkConfig)
	if ok {
		setStringIfNotNil(c.WhiteList, &other.NICWhiteList)
		setStringIfNotNil(c.BlackList, &other.NICBlackList)
	}
	return ok
}

type MSSQLConfig struct {
	EnabledClasses *string `yaml:"enabled_classes"`
}

func (c *MSSQLConfig) translate(cm map[string]string) {
	setIfNotNil(cm, "collectors.mssql.classes-enabled", c.EnabledClasses)
}

func (c *MSSQLConfig) Sync(v interface{}) bool {
	other, ok := v.(*collector.MSSQLConfig)
	if ok {
		setStringIfNotNil(c.EnabledClasses, &other.MSSQLEnabledCollectors)
	}
	return ok
}

type MSMQConfig struct {
	Where *string `yaml:"where_clause"`
}

func (c *MSMQConfig) translate(cm map[string]string) {
	setIfNotNil(cm, "collector.msmq.msmq-where", c.Where)
}

func (c *MSMQConfig) Sync(v interface{}) bool {
	other, ok := v.(*collector.MSMQConfig)
	if ok {
		setStringIfNotNil(c.Where, &other.MSMQWhereClause)
	}
	return ok
}

type LogicalDiskConfig struct {
	WhiteList *string `yaml:"whitelist"`
	BlackList *string `yaml:"blacklist"`
}

func (c *LogicalDiskConfig) translate(cm map[string]string) {
	setIfNotNil(cm, "collector.logical_disk.volume-whitelist", c.WhiteList)
	setIfNotNil(cm, "collector.logical_disk.volume-blacklist", c.BlackList)
}

func (c *LogicalDiskConfig) Sync(v interface{}) bool {
	other, ok := v.(*collector.LogicalDiskConfig)
	if ok {
		setStringIfNotNil(c.WhiteList, &other.VolumeWhiteList)
		setStringIfNotNil(c.BlackList, &other.VolumeBlackList)
	}
	return ok
}

type translatableConfig interface {
	translate(cm map[string]string)
	Sync(v interface{}) bool
}

func translateConfig(c translatableConfig, cm map[string]string) {
	if c == nil || reflect.ValueOf(c).IsNil() {
		return
	}
	c.translate(cm)
}

func setIfNotNil(cm map[string]string, key string, value *string) {
	if value == nil {
		return
	}
	cm[key] = *value
}

func setStringIfNotNil(source *string, destination *string) {
	if source == nil {
		return
	}
	*destination = *source
}
