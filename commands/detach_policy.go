package commands

import (
	"errors"
	"os"

	"code.cloudfoundry.org/app-autoscaler-cli-plugin/api"
	"code.cloudfoundry.org/app-autoscaler-cli-plugin/ui"
)

type DetachPolicyCommand struct {
	RequiredlArgs DetachPolicyPositionalArgs `positional-args:"yes"`
}

type DetachPolicyPositionalArgs struct {
	AppName string `positional-arg-name:"APP_NAME" required:"true" `
}

func (command DetachPolicyCommand) Execute([]string) error {
	return DetachPolicy(AutoScaler.CLIConnection, command.RequiredlArgs.AppName, AutoScaler.UserAgent)
}

func DetachPolicy(cliConnection api.Connection, appName string, userAgent string) error {

	cfclient, err := api.NewCFClient(cliConnection, userAgent)
	if err != nil {
		return err
	}

	endpoint, err := api.GetEndpoint(cfclient, userAgent)
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

	apihelper := api.NewAPIHelper(endpoint, cfclient, os.Getenv("CF_TRACE"), userAgent)

	ui.SayMessage(ui.DetachPolicyHint, appName)
	err = apihelper.DeletePolicy()
	if err != nil {
		return err
	}

	ui.SayOK()
	return nil
}
