package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"code.cloudfoundry.org/cli/cf/configuration/confighelpers"

	"code.cloudfoundry.org/app-autoscaler-cli-plugin/ui"
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

func UnsetEndpoint() error {

	configFilePath := ConfigFile()
	err := ioutil.WriteFile(configFilePath, nil, 0600)
	if err != nil {
		return err
	}
	return nil
}

func SetEndpoint(cfclient *CFClient, url string, skipSSLValidation bool) error {

	cfDomain := getDomain(cfclient.CCAPIEndpoint)
	autoscalerDomain := getDomain(url)
	if cfDomain != autoscalerDomain {
		return fmt.Errorf(ui.InconsistentDomain, url, cfclient.CCAPIEndpoint)
	}

	skipSSLValidation = skipSSLValidation || cfclient.IsSSLDisabled
	endpoint := &APIEndpoint{
		URL:               strings.TrimSuffix(url, "/"),
		SkipSSLValidation: skipSSLValidation,
	}

	apihelper := NewAPIHelper(endpoint, cfclient, os.Getenv("CF_TRACE"))
	err := apihelper.CheckHealth()
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

func GetEndpoint(cfclient *CFClient) (*APIEndpoint, error) {

	endpoint, err := getEndpointFromConfig()
	if err != nil {
		return nil, err
	}

	if endpoint.URL != "" {
		cfDomain := getDomain(cfclient.CCAPIEndpoint)
		autoscalerDomain := getDomain(endpoint.URL)
		if cfDomain != autoscalerDomain {
			UnsetEndpoint()
			endpoint = &APIEndpoint{}
		}
	}

	if endpoint.URL == "" {
		endpoint, err = getDefaultEndpoint(cfclient)
		if err != nil {
			return nil, err
		}
	}
	return endpoint, nil

}

func getDefaultEndpoint(cfclient *CFClient) (*APIEndpoint, error) {

	ccAPIURL := cfclient.CCAPIEndpoint
	asAPIURL := strings.Replace(ccAPIURL, "api.", "autoscaler.", 1)

	//ignore all erros here if the default value won't work
	SetEndpoint(cfclient, asAPIURL, cfclient.IsSSLDisabled)
	return getEndpointFromConfig()

}

func getEndpointFromConfig() (*APIEndpoint, error) {

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

func getDomain(urlstr string) string {

	domain := ""
	if strings.HasSuffix(urlstr, "/") {
		urlstr = strings.TrimSuffix(urlstr, "/")
	}
	if !strings.HasPrefix(urlstr, "http") {
		urlstr = "https://" + urlstr
	}

	u, err := url.Parse(urlstr)
	if err == nil && strings.Contains(u.Hostname(), ".") {
		domain = strings.SplitN(u.Hostname(), ".", 2)[1]
	}

	return domain

}
