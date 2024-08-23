package api

import (
	"context"
	"fmt"

	plugin_models "code.cloudfoundry.org/cli/plugin/models"
	cf_client "github.com/cloudfoundry/go-cfclient/v3/client"
	cf_client_config "github.com/cloudfoundry/go-cfclient/v3/config"
	"github.com/cloudfoundry/go-cfclient/v3/resource"

	"code.cloudfoundry.org/app-autoscaler-cli-plugin/ui"
)

type CFClient struct {
	connection    Connection
	CCAPIEndpoint string
	AuthToken     string
	AppId         string
	AppName       string
	IsSSLDisabled bool
}

type Connection interface {
	ApiEndpoint() (string, error)
	HasSpace() (bool, error)
	IsLoggedIn() (bool, error)
	AccessToken() (string, error)
	GetCurrentSpace() (plugin_models.Space, error)
	IsSSLDisabled() (bool, error)
}

func NewCFClient(connection Connection) (*CFClient, error) {

	ccAPIEndpoint, err := connection.ApiEndpoint()
	if err != nil {
		return nil, err
	}
	if ccAPIEndpoint == "" {
		return nil, fmt.Errorf(ui.NOCFAPIEndpoint)
	}

	isSSLDisabled, err := connection.IsSSLDisabled()
	if err != nil {
		return nil, err
	}

	client := &CFClient{
		connection:    connection,
		CCAPIEndpoint: ccAPIEndpoint,
		IsSSLDisabled: isSSLDisabled,
	}

	return client, nil

}

func (client *CFClient) Configure(appName string) error {

	if connected, err := client.connection.IsLoggedIn(); !connected {
		if err != nil {
			return err
		}
		return fmt.Errorf(ui.LoginRequired, client.CCAPIEndpoint)
	}

	if hasSpace, err := client.connection.HasSpace(); !hasSpace {
		if err != nil {
			return err
		}
		return fmt.Errorf(ui.NoTarget)
	}

	currentSpace, err := client.connection.GetCurrentSpace()
	if err != nil {
		return err
	}

	authToken, err := client.connection.AccessToken()
	if err != nil {
		return err
	}

	app, err := GetApp(client.CCAPIEndpoint, authToken, appName, currentSpace.Guid)
	if err != nil {
		return err
	}

	client.AuthToken = authToken
	client.AppId = app.GUID
	client.AppName = appName
	return nil

}

func GetApp(ccAPIEndpoint string, authToken string, appName string, currentSpaceGUID string) (*resource.App, error) {
	// A refresh token is not provided by the CF CLI Plugin API and is not required as
	// "AccessToken() now provides a refreshed o-auth token.",
	// see https://github.com/cloudfoundry/cli/blob/main/plugin/plugin_examples/CHANGELOG.md#changes-in-v614
	refreshToken := ""

	cfg, err := cf_client_config.New(ccAPIEndpoint, cf_client_config.Token(authToken, refreshToken))
	if err != nil {
		return nil, err
	}
	cf, err := cf_client.New(cfg)
	if err != nil {
		return nil, err
	}

	appFilter := &cf_client.AppListOptions{
		Names:      cf_client.Filter{Values: []string{appName}},
		SpaceGUIDs: cf_client.Filter{Values: []string{currentSpaceGUID}},
	}
	apps, err := cf.Applications.ListAll(context.Background(), appFilter)
	if err != nil {
		return nil, err
	}

	if len(apps) == 0 {
		return nil, fmt.Errorf(ui.NoApp, appName)
	}

	app := apps[0]
	return app, nil
}
