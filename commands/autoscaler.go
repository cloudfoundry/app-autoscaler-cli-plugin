package commands

import (
	"code.cloudfoundry.org/app-autoscaler-cli-plugin/api"
)

type AutoScalerCmds struct {
	CLIConnection api.Connection

	API              ApiCommand              `command:"autoscaling-api" description:"Set or view AutoScaler service API endpoint"`
	Policy           PolicyCommand           `command:"autoscaling-policy" description:"Retrieve the scaling policy of an application"`
	AttachPolicy     AttachPolicyCommand     `command:"attach-autoscaling-policy" description:"Attach a scaling policy to an application"`
	DetachPolicy     DetachPolicyCommand     `command:"detach-autoscaling-policy" description:"Detach a scaling policy from an application"`
	CreateCredential CreateCredentialCommand `command:"create-autoscaling-credential" description:"Create custom metric credential for an application"`
	DeleteCredential DeleteCredentialCommand `command:"delete-autoscaling-credential" description:"Delete the custom metric credential of an application"`
	Metrics          MetricsCommand          `command:"autoscaling-metrics" description:"Retrieve the metrics of an application"`
	History          HistoryCommand          `command:"autoscaling-history" description:"Retrieve the history of an application"`

	UninstallPlugin UninstallHook `command:"CLI-MESSAGE-UNINSTALL"`
}

var AutoScaler AutoScalerCmds
