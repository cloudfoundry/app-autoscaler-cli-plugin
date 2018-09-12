package commands

import (
	"strings"

	"cli/api"
	"cli/ui"
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
		return cmd.GetEndpoint()
	} else {
		return cmd.SetEndpoint(AutoScaler.CLIConnection, cmd.OptionalArgs.URL, cmd.SkipSSLValidation)
	}
}

func (cmd *ApiCommand) GetEndpoint() error {

	endpoint, err := api.GetEndpoint()
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

	if strings.HasSuffix(url, "/") {
		url = strings.TrimSuffix(url, "/")
	}
	if !strings.HasPrefix(url, "http") {
		url = "https://" + url
	}

	ui.SayMessage(ui.SetAPIEndpoint, url)

	err := api.SetEndpoint(cliConnection, url, skipSSLValidation)
	if err != nil {
		return err
	}
	ui.SayOK()
	return nil

}
