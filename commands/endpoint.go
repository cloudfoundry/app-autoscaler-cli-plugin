package commands

import (
	"strings"

	"code.cloudfoundry.org/app-autoscaler-cli-plugin/api"
	"code.cloudfoundry.org/app-autoscaler-cli-plugin/ui"
)

type ApiCommand struct {
	OptionalArgs      APIPositionalArgs `positional-args:"yes"`
	Unset             bool              `long:"unset" description:"Unset the api endpoint"`
	SkipSSLValidation bool              `long:"skip-ssl-validation" description:"Skip verification of the API endpoint. Not recommended!"`
}

type APIPositionalArgs struct {
	URL string `positional-arg-name:"URL" description:"The autoscaler API endpoint"`
}

func (cmd ApiCommand) Execute([]string) error {

	if cmd.Unset {
		return cmd.UnsetEndpoint()
	}
	if cmd.OptionalArgs.URL == "" {
		return cmd.GetEndpoint(AutoScaler.CLIConnection)
	} else {
		return cmd.SetEndpoint(AutoScaler.CLIConnection, cmd.OptionalArgs.URL, cmd.SkipSSLValidation)
	}
}

func (cmd *ApiCommand) GetEndpoint(cliConnection api.Connection) error {

	cfclient, err := api.NewCFClient(cliConnection)
	if err != nil {
		return err
	}
	endpoint, err := api.GetEndpoint(cfclient)
	if err != nil {
		return err
	}

	if endpoint.URL == "" {
		ui.SayMessage(ui.NoEndpoint)
	} else {
		ui.SayMessage(ui.APIEndpoint, endpoint.URL)
	}
	return nil
}

func (cmd *ApiCommand) UnsetEndpoint() error {

	ui.SayMessage(ui.UnsetAPIEndpoint)

	err := api.UnsetEndpoint()
	if err != nil {
		return err
	}
	ui.SayOK()
	return nil

}

func (cmd *ApiCommand) SetEndpoint(cliConnection api.Connection, url string, skipSSLValidation bool) error {

	cfclient, err := api.NewCFClient(cliConnection)
	if err != nil {
		return err
	}

	if strings.HasSuffix(url, "/") {
		url = strings.TrimSuffix(url, "/")
	}
	if !strings.HasPrefix(url, "http") {
		url = "https://" + url
	}

	ui.SayMessage(ui.SetAPIEndpoint, url)
	err = api.SetEndpoint(cfclient, url, skipSSLValidation)
	if err != nil {
		return err
	}
	ui.SayOK()
	return nil

}
