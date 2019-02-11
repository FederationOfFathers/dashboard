package config

import (
	"fmt"
	"io/ioutil"
	"os"

	"go.uber.org/zap"
	yaml "gopkg.in/yaml.v2"
)

var Logger *zap.Logger

// DiscordConfig holds configuratioins information for Discord API Integration
// We could probably change this to be different for multiple Discord bots/apps
var DiscordConfig *DiscordCfg

func init() {

	err := unmarshalConfig("cfg-discord.yml", &DiscordConfig)
	if err != nil {
		fmt.Printf("Unable to load Discord config. %s", err)
		DiscordConfig = nil
	}
}

// UnmarshalConfig unmarshals a config YML file into an interface
func unmarshalConfig(fileName string, cfgObject interface{}) error {
	// exit quietly if no file. assume we are not configuring that portion
	if _, err := os.Stat(fileName); err != nil {
		return err
	}

	// read file data
	fileData, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}

	// unmarshal into interface object
	err2 := yaml.Unmarshal(fileData, cfgObject)
	if err2 != nil {
		return err2
	}

	return nil
}
