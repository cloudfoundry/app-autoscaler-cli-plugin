package commands

import (
	"cli/api"
	"cli/ui"
	"errors"
	"os"
)

type DeleteCredentialCommand struct {
	RequiredlArgs DeleteCredentialPositionalArgs `positional-args:"yes"`
}

type DeleteCredentialPositionalArgs struct {
	AppName string `positional-arg-name:"APP_NAME" required:"true" `
}

func (command DeleteCredentialCommand) Execute([]string) error {
	return DeleteCredential(AutoScaler.CLIConnection, command.RequiredlArgs.AppName)
}

func DeleteCredential(cliConnection api.Connection, appName string) error {

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

	ui.SayMessage(ui.DeleteCredentialHint, appName)
	err = apihelper.DeleteCredential()
	if err != nil {
		return err
	}

	ui.SayOK()
	ui.SayWarningMessage(ui.DeleteCredentialWarning, appName)
	return nil
}
