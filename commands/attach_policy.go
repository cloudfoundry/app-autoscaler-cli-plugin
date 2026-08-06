package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"code.cloudfoundry.org/app-autoscaler-cli-plugin/api"
	"code.cloudfoundry.org/app-autoscaler-cli-plugin/ui"
)

type AttachPolicyCommand struct {
	RequiredlArgs AttachPolicyPositionalArgs `positional-args:"yes"`
}

type AttachPolicyPositionalArgs struct {
	AppName    string `positional-arg-name:"APP_NAME" required:"true" `
	PolicyFile string `positional-arg-name:"PATH_TO_POLICY_FILE" required:"true"`
}

func (command AttachPolicyCommand) Execute([]string) error {
	return CreatePolicy(AutoScaler.CLIConnection, command.RequiredlArgs.AppName, command.RequiredlArgs.PolicyFile, AutoScaler.UserAgent)
}

func CreatePolicy(cliConnection api.Connection, appName string, policyFile string, userAgent string) error {

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

	ui.SayMessage(ui.AttachPolicyHint, appName)
	contents, err := ioutil.ReadFile(policyFile)
	if err != nil {
		return fmt.Errorf(ui.FailToLoadPolicyFile, policyFile)
	}
	var policy map[string]interface{}
	err = json.Unmarshal(contents, &policy)
	if err != nil {
		return fmt.Errorf(ui.InvalidPolicy, err)
	}

	err = apihelper.CreatePolicy(policy)
	if err != nil {
		return err
	}

	ui.SayOK()
	return nil
}
