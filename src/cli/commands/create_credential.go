package commands

import (
	"cli/api"
	"cli/ui"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"io"
	"os"
)

type CreateCredentialCommand struct {
	PositionalArgs CreateCredentialPositionalArgs `positional-args:"yes"`
	Output         string                         `long:"output" description:"dump the policy to a file in JSON format"`
}

type CreateCredentialPositionalArgs struct {
	AppName        string `positional-arg-name:"APP_NAME" required:"true" `
	CredentialFile string `positional-arg-name:"CREDENTIAL_FILE"`
}

func (command CreateCredentialCommand) Execute([]string) error {

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

	return CreateCredential(AutoScaler.CLIConnection, command.PositionalArgs.AppName, command.PositionalArgs.CredentialFile, writer, command.Output)
}

func CreateCredential(cliConnection api.Connection, appName string, credentialFile string, writer io.Writer, outputfile string) error {

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
		ui.SayMessage(ui.SaveCredentialHint, appName, outputfile)
	} else {
		ui.SayMessage(ui.CreateCredentialHint, appName)
	}

	var credentialSource map[string]interface{}
	var credentialResult []byte
	if credentialFile != "" {
		contents, err := ioutil.ReadFile(credentialFile)
		if err != nil {
			return fmt.Errorf(ui.FailToLoadCredentialFile, credentialFile)
		}
		err = json.Unmarshal(contents, &credentialSource)
		if err != nil {
			return fmt.Errorf(ui.InvalidCredential, err)
		}
		credentialResult, err = apihelper.CreateCredential(credentialSource)
		if err != nil {
			return err
		}
	} else {
		credentialResult, err = apihelper.CreateCredential(nil)
		if err != nil {
			return err
		}
	}
	fmt.Fprintf(writer, "%v", string(credentialResult))

	if outputfile != "" {
		ui.SayOK()
	}
	return nil
}
