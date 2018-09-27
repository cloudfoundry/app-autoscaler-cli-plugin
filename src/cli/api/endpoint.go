package api

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"code.cloudfoundry.org/cli/cf/configuration/confighelpers"
)

type APIEndpoint struct {
	URL               string
	SkipSSLValidation bool
}

var ConfigFile = func() string {
	defaultCFConfigPath, _ := confighelpers.DefaultFilePath()
	targetsPath := filepath.Join(filepath.Dir(defaultCFConfigPath), "plugins", "autoscaler_config")
	os.MkdirAll(targetsPath, 0700)

	defaultConfigFileName := "config.json"
	if os.Getenv("AUTOSCALER_CONFIG_FILE") != "" {
		defaultConfigFileName = os.Getenv("AUTOSCALER_CONFIG_FILE")
	}
	return filepath.Join(targetsPath, defaultConfigFileName)
}

func GetEndpoint() (*APIEndpoint, error) {

	configFilePath := ConfigFile()
	endpoint := &APIEndpoint{}

	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		err := ioutil.WriteFile(configFilePath, nil, 0600)
		if err != nil {
			return nil, err
		}
	} else {
		content, err := ioutil.ReadFile(configFilePath)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(content, &endpoint)
		if err != nil || endpoint.URL == "" {
			ioutil.WriteFile(configFilePath, nil, 0600)
		}
	}
	return endpoint, nil

}

func UnsetEndpoint() error {

	configFilePath := ConfigFile()
	err := ioutil.WriteFile(configFilePath, nil, 0600)
	if err != nil {
		return err
	}
	return nil
}

func SetEndpoint(cliConnection Connection, url string, skipSSLValidation bool) error {

	cfclient, err := NewCFClient(cliConnection)
	if err != nil {
		return err
	}
	skipSSLValidation = skipSSLValidation || cfclient.IsSSLDisabled
	endpoint := &APIEndpoint{
		URL:               strings.TrimSuffix(url, "/"),
		SkipSSLValidation: skipSSLValidation,
	}

	apihelper := NewAPIHelper(endpoint, cfclient, os.Getenv("CF_TRACE"))
	err = apihelper.CheckHealth()
	if err != nil {
		return err
	}

	urlConfig, err := json.Marshal(endpoint)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(ConfigFile(), urlConfig, 0600)
	if err != nil {
		return err
	}

	return nil
}
