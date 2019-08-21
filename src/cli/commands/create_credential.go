package commands

import (
	"cli/api"
	"cli/ui"
	"errors"
	"fmt"
	"io"
	"os"

	"cli/models"
)

type CreateCredentialCommand struct {
	RequiredArgs CreateCredentialPositionalArgs `positional-args:"yes"`
	Username     string                         `short:"u" long:"username" description:"username of the custom metric credential, random username will be set if not specified"`
	Password     string                         `short:"p" long:"password" description:"password of the custom metric credential, random password will be set if not specified"`
	Output       string                         `long:"output" description:"dump the credential to a file in JSON format"`
}

type CreateCredentialPositionalArgs struct {
	AppName        string `positional-arg-name:"APP_NAME" required:"true" `
}

func (command CreateCredentialCommand) Execute([]string) error {

	var (
		err    error
		writer *os.File
	)

	if command.Username == "" && command.Password != "" {
		return fmt.Errorf(ui.InvalidCredentialUsername)
	} else if command.Username != "" && command.Password == "" {
		return fmt.Errorf(ui.InvalidCredentialPassword)
	}

	if command.Output != "" {
		writer, err = os.OpenFile(command.Output, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
		defer writer.Close()
	} else {
		writer = os.Stdout
	}

	return CreateCredential(AutoScaler.CLIConnection, command.RequiredArgs.AppName, command.Username, command.Password, writer, command.Output)
}

func CreateCredential(cliConnection api.Connection, appName string, username string, password string, writer io.Writer, outputfile string) error {

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

	var credentialResult []byte
	if username != "" && password != "" {
		credentialSource := models.Credential {
			Username: username,
			Password: password,
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
