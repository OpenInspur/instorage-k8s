package utils

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/go-yaml/yaml"
)

//LogCfg contain configuration about log set.
type LogCfg struct {
	Enabled          bool
	LogDir           string
	Level            string
	LogRotateMaxSize int
}

//HostCfg contain configuration about host.
type HostCfg struct {
	//iscsi or fc
	Link              string
	ForceUseMultiPath bool `yaml:"forceUseMultipath"`

	SCSIScanRetryTimes   int `yaml:"scsiScanRetryTimes"`
	SCSIScanWaitInterval int `yaml:"scsiScanWaitInterval"`

	ISCSIPathCheckRetryTimes   int `yaml:"iscsiPathCheckRetryTimes"`
	ISCSIPathCheckWaitInterval int `yaml:"iscsiPathCheckWaitInterval"`

	MultiPathSearchRetryTimes   int `yaml:"multipathSearchRetryTimes"`
	MultiPathSearchWaitInterval int `yaml:"multipathSearchWaitInterval"`
	MultiPathResizeDelay        int `yaml:"multipathResizeDelay"`
}

//StorageCfg contain configuration about storage access.
type StorageCfg struct {
	Name           string
	StrType        string `yaml:"type"`
	Host           string
	Username       string
	Password       string
	Shadow         string
	DeviceUsername string `yaml:"deviceUsername"`
	DevicePassword string `yaml:"devicePassword"`
	DeviceShadow   string `yaml:"deviceShadow"`
	BarrierPath    string `yaml:"barrierPath"`
}

//Config contain the configuration from configure file.
//Configure file is in the 'config' folder as name 'instorage.yaml'.
//The 'config' folder is in the same folder as the driver.
//The contant of config file is yaml format.
type Config struct {
	Log     LogCfg
	Host    HostCfg
	Storage []StorageCfg
}

//CheckConfig make a check of the configuration.
func (c *Config) CheckConfig() error {

	if c.Host.Link != "iscsi" && c.Host.Link != "fc" {
		return fmt.Errorf("host.link can be iscsi or fc and can only be iscsi for AS13000 storage")
	}

	if len(c.Storage) == 0 {
		return fmt.Errorf("storage must configured")
	}

	for i, s := range c.Storage {
		if strings.ToUpper(s.StrType) != "AS18000" && strings.ToUpper(s.StrType) != "AS13000" {
			return fmt.Errorf("storage type must be AS13000 or AS18000, current:%s", s.StrType)
		}
		if strings.ToUpper(s.StrType) == "AS13000" {
			if c.Host.Link != "iscsi" {
				return fmt.Errorf("host.link must be iscsi")
			}

			if s.Name == "" || s.Host == "" || s.Username == "" || s.Password == "" || s.DeviceUsername == "" || s.DevicePassword == "" {
				return fmt.Errorf("storage %d configure not complete, name, host, username, password, deviceUsername, devicePassword must be set", i)
			}

		} else {
			if s.Name == "" || s.Host == "" || s.Username == "" || s.Password == "" {
				return fmt.Errorf("storage %d configure not complete, name, host, username, password or shadow must be set", i)
			}

		}

	}

	return nil
}

func (c *Config) setDefault(baseDir string) {
	if c.Log.LogDir == "" {
		c.Log.LogDir = "log"
	}

	if c.Host.ISCSIPathCheckRetryTimes == 0 {
		c.Host.ISCSIPathCheckRetryTimes = 3
	}

	if c.Host.ISCSIPathCheckWaitInterval == 0 {
		c.Host.ISCSIPathCheckWaitInterval = 1
	}

	if c.Host.MultiPathSearchRetryTimes == 0 {
		c.Host.MultiPathSearchRetryTimes = 3
	}

	if c.Host.MultiPathSearchWaitInterval == 0 {
		c.Host.MultiPathSearchWaitInterval = 1
	}

	if c.Host.MultiPathResizeDelay == 0 {
		c.Host.MultiPathResizeDelay = 1
	}

	if c.Host.SCSIScanRetryTimes == 0 {
		c.Host.SCSIScanRetryTimes = 3
	}

	if c.Host.SCSIScanWaitInterval == 0 {
		c.Host.SCSIScanWaitInterval = 1
	}

	//Set default barrier path, so if not set, barrier file will
	//in the same foler as the exe.
	for idx, str := range c.Storage {
		if str.BarrierPath == "" {
			c.Storage[idx].BarrierPath = baseDir
		}
	}
}

func (c *Config) decryptShadow() error {
	cipher := Cipher{}

	for idx, str := range c.Storage {
		if str.Password == "" && str.Shadow != "" {
			c.Storage[idx].Password = cipher.Decrypt(str.Shadow)
		}
		if str.DevicePassword == "" && str.DeviceShadow != "" {
			c.Storage[idx].DevicePassword = cipher.Decrypt(str.DeviceShadow)
		}
	}

	return nil
}

//LoadConfig load configuration from the config file.
func LoadConfig(configPath string, baseDir string) (*Config, error) {
	var config Config

	cfgData, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config file %s failed for %s", configPath, err)
	}

	if err := yaml.Unmarshal(cfgData, &config); err != nil {
		return nil, fmt.Errorf("parse config file %s failed for %s", cfgData, err)
	}

	config.setDefault(baseDir)

	if err := config.decryptShadow(); err != nil {
		return nil, fmt.Errorf("shadow decrypt failed %s", err)
	}

	if err := config.CheckConfig(); err != nil {
		return nil, fmt.Errorf("configuration not valied %s", err)
	}

	return &config, nil
}

//Dump18000SampleConfig generate a 18000 storage sample configuration file's content.
func Dump18000SampleConfig() string {
	config := Config{}

	//set the default.
	config.setDefault("<path/to/bin/folder>")

	//set the sample host link.
	config.Host.Link = "iscsi"

	//add a sample storage configuration.
	strCfg := StorageCfg{
		Name:     "storage-01",
		StrType:  "AS18000",
		Host:     "10.0.0.1:22",
		Username: "username",
		Password: "password",
		Shadow:   "<shadow of the password, can be generated by instorage flexvolume driver like './instorage ext-encrypt-password [password]'>",
	}
	config.Storage = append(config.Storage, strCfg)

	//marshal the data.
	cfgData, _ := yaml.Marshal(config)
	return string(cfgData)
}

//Dump13000SampleConfig generate a 13000 storage sample configuration file's content.
func Dump13000SampleConfig() string {
	config := Config{}

	//set the default.
	config.setDefault("<path/to/bin/folder>")

	//set the sample host link.
	config.Host.Link = "iscsi"

	//add a sample storage configuration.
	strCfg := StorageCfg{
		Name:           "storage-01",
		StrType:        "AS13000",
		Host:           "10.0.0.1:8080",
		Username:       "admin",
		Password:       "passw0rd",
		Shadow:         "<shadow of the password, can be generated by instorage flexvolume driver like './instorage ext-encrypt-password [password]'>",
		DeviceUsername: "superuser",
		DevicePassword: "00000000",
	}
	config.Storage = append(config.Storage, strCfg)

	//marshal the data.
	cfgData, _ := yaml.Marshal(config)
	return string(cfgData)
}
