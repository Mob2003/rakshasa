package server

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"rakshasa_lite/common"
)

var currentConfig common.Config

func SetConfig(config common.Config) {
	currentConfig = config
	currentConfig.FileSave = false
	currentNode.mainIp = currentConfig.ListenIp
	currentNode.port = currentConfig.Port
}
func ConfigSave() error {
	b, _ := yaml.Marshal(currentConfig)
	err := ioutil.WriteFile(currentConfig.FileName, b, 0666)
	if err == nil {
		currentConfig.FileSave = true
	}
	return err
}
func ConfigLoad(filename string) error {
	b, err := ioutil.ReadFile(filename)
	if err == nil {
		var config common.Config
		err = yaml.Unmarshal(b, &config)
		if err == nil {

			currentConfig = config
			currentConfig.FileSave = true
		}
	}

	return err
}
func GetConfig() common.Config {

	return currentConfig
}
