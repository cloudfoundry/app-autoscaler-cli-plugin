package commands

import (
	"cli/api"
	"cli/ui"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"cli/models"
)

type AttachPolicyCommand struct {
	RequiredlArgs AttachPolicyPositionalArgs `positional-args:"yes"`
}

type AttachPolicyPositionalArgs struct {
	AppName    string `positional-arg-name:"APP_NAME" required:"true" `
	PolicyFile string `positional-arg-name:"PATH_TO_POLICY_FILE" required:"true"`
}

func (command AttachPolicyCommand) Execute([]string) error {
	return CreatePolicy(AutoScaler.CLIConnection, command.RequiredlArgs.AppName, command.RequiredlArgs.PolicyFile)
}

func CreatePolicy(cliConnection api.Connection, appName string, policyFile string) error {

	cfclient, err := api.NewCFClient(cliConnection)
	if err != nil {
		return err
	}
	err = cfclient.Configure(appName)
	if err != nil {
		return err
	}

	endpoint, err := api.GetEndpoint()
	if err != nil {
		return err
	}
	if endpoint.URL == "" {
		return errors.New(ui.NoEndpoint)
	}
	apihelper := api.NewAPIHelper(endpoint, cfclient, os.Getenv("CF_TRACE"))

	ui.SayMessage(ui.AttachPolicyHint, appName)
	contents, err := ioutil.ReadFile(policyFile)
	if err != nil {
		return fmt.Errorf(ui.FailToLoadPolicyFile, policyFile)
	}

	var policy models.ScalingPolicy
	err = json.Unmarshal(contents, &policy)
	if err != nil {
		return fmt.Errorf(ui.InvalidPolicy, err)
	}

	err = apihelper.CreatePolicy(policy)
	if err != nil {
		return err
	}
	ui.SayOK()

	if policy.Schedules != nil {
		warning := false

		if policy.Schedules.RecurringSchedules != nil {
			if policy.Schedules.SpecificDateSchedules != nil {
				warning = true
			} else {
				hasDayofMonth := false
				hasDayofWeek := false
				for _, schedule := range policy.Schedules.RecurringSchedules {
					if schedule.DaysOfMonth != nil {
						hasDayofMonth = true
					}
					if schedule.DaysOfWeek != nil {
						hasDayofWeek = true
					}
				}
				warning = hasDayofMonth && hasDayofWeek
			}
		}
		if warning {
			ui.SayMessage(ui.ScheduleConflictWarning)
		}

	}

	return nil
}
