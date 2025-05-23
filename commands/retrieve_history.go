package commands

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler-cli-plugin/api"
	"code.cloudfoundry.org/app-autoscaler-cli-plugin/ui"
	ctime "code.cloudfoundry.org/app-autoscaler-cli-plugin/util/time"
)

type HistoryCommand struct {
	RequiredlArgs HistoryPositionalArgs `positional-args:"yes"`
	StartTime     string                `long:"start" description:"start time of metrics collected with format \"yyyy-MM-ddTHH:mm:ss+/-HH:mm\" or \"yyyy-MM-ddTHH:mm:ssZ\", default to very beginning if not specified."`
	EndTime       string                `long:"end" description:"end time of the metrics collected with format \"yyyy-MM-ddTHH:mm:ss+/-HH:mm\" or \"yyyy-MM-ddTHH:mm:ssZ\", default to current time if not speficied."`
	Desc          bool                  `long:"desc" description:"display in descending order, default to ascending order if not specified."`
	Asc           bool                  `long:"asc" description:"display in ascending order, default to descending order if not specified."`
	Output        string                `long:"output" description:"dump the policy to a file in JSON format"`
}

type HistoryPositionalArgs struct {
	AppName string `positional-arg-name:"APP_NAME" required:"true"`
}

func (command HistoryCommand) Execute([]string) error {

	var (
		st     int64 = 0
		et     int64 = time.Now().UnixNano()
		fpo    bool  = false
		err    error
		writer *os.File
	)
	if command.Desc && command.Asc {
		return fmt.Errorf(ui.DeprecatedDescWarning)
	}
	if command.StartTime != "" {
		st, err = ctime.ParseTimeFormat(command.StartTime)
		if err != nil {
			return err
		}
	}
	if command.EndTime != "" {
		et, err = ctime.ParseTimeFormat(command.EndTime)
		if err != nil {
			return err
		}
	}
	if st > et {
		return errors.New(fmt.Sprintf(ui.InvalidTimeRange, command.StartTime, command.EndTime))
	}
	fpo = command.StartTime == "" && command.EndTime == ""

	if command.Output != "" {
		writer, err = os.OpenFile(command.Output, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
		defer writer.Close()
	} else {
		writer = os.Stdout
	}

	return RetrieveHistory(AutoScaler.CLIConnection,
		command.RequiredlArgs.AppName,
		st, et, fpo, command.Desc, command.Asc, writer, command.Output)
}

func RetrieveHistory(cliConnection api.Connection, appName string, startTime, endTime int64, firstPageOnly bool, desc bool, asc bool, writer io.Writer, outputfile string) error {

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
		ui.SayMessage(ui.SaveHistoryHint, appName, outputfile)
	} else {
		ui.SayMessage(ui.ShowHistoryHint, appName)
	}

	table := ui.NewTable(writer, []string{"Scaling Type", "Status", "Instance Changes", "Time", "Action", "Error"})
	var (
		page       uint64 = 1
		next       bool   = true
		noResult   bool   = true
		moreResult bool   = false
		data       [][]string
	)

	for {
		next, data, err = apihelper.GetHistory(startTime, endTime, asc, page)
		if err != nil {
			return err
		}

		for _, row := range data {
			table.Add(row)
		}
		if len(data) > 0 {
			noResult = false
			table.Print()
		}

		moreResult = next && firstPageOnly
		if !next || firstPageOnly {
			break
		}
		page += 1
	}

	if noResult {
		ui.SayOK()
		ui.SayMessage(ui.HistoryNotFound, appName)
	} else {
		if outputfile != "" {
			ui.SayOK()
		}
	}
	if moreResult {
		ui.SayWarningMessage(ui.MoreRecordsWarning)
	}
	if desc {
		ui.SayWarningMessage(ui.DeprecatedDescWarning)
	}

	return nil
}
