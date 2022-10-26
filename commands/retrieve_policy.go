package commands

import (
	"code.cloudfoundry.org/app-autoscaler-cli-plugin/ui"
	"code.cloudfoundry.org/app-autoscaler-cli-plugin/api"
	"errors"
	"fmt"
	"io"
	"os"
)

type PolicyCommand struct {
	RequiredlArgs PolicyPositionalArgs `positional-args:"yes"`
	Output        string               `long:"output" description:"dump the policy to a file in JSON format"`
}

type PolicyPositionalArgs struct {
	AppName string `positional-arg-name:"APP_NAME" required:"true"`
}

func (command PolicyCommand) Execute([]string) error {

	var (
		err    error
		writer *os.File
	)

	if command.Output != "" {
		writer, err = os.OpenFile(command.Output, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
		defer writer.Close()
	} else {
		writer = os.Stdout
	}

	return RetrievePolicy(AutoScaler.CLIConnection, command.RequiredlArgs.AppName, writer, command.Output)
}

func RetrievePolicy(cliConnection api.Connection, appName string, writer io.Writer, outputfile string) error {

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

	if outputfile != "" {
		ui.SayMessage(ui.SavePolicyHint, appName, outputfile)
	} else {
		ui.SayMessage(ui.ShowPolicyHint, appName)
	}

	policy, err := apihelper.GetPolicy()
	if err != nil {
		return err
	}
	fmt.Fprintf(writer, "%v", string(policy))

	if outputfile != "" {
		ui.SayOK()
	}
	return nil
}
