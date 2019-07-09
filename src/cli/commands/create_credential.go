package commands

import (
	"cli/api"
	"cli/ui"
	"errors"
	"os"
)

type CreateCredentialCommand struct {
	RequiredlArgs CreateCredentialPositionalArgs `positional-args:"yes"`
}

type CreateCredentialPositionalArgs struct {
	AppName string `positional-arg-name:"APP_NAME" required:"true" `
}

func (command CreateCredentialCommand) Execute([]string) error {
	return CreateCredential(AutoScaler.CLIConnection, command.RequiredlArgs.AppName)
}

func CreateCredential(cliConnection api.Connection, appName string) error {

	cfclient, err := api.NewCFClient(cliConnection)
	if err != nil {
		return err
	}
	endpoint, err := api.GetEndpoint(cfclient)
	if err != nil {
		return err
	}
	if endpoint.URL == "" {
		return errors.New(ui.NoEndpoint)
	}

	err = cfclient.Configure(appName)
	if err != nil {
		return err
	}

	apihelper := api.NewAPIHelper(endpoint, cfclient, os.Getenv("CF_TRACE"))

	ui.SayMessage(ui.CreateCredentialHint, appName)
	err = apihelper.CreateCredential()
	if err != nil {
		return err
	}

	ui.SayOK()
	return nil
}
