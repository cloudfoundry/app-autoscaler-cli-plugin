package commands

import (
	"cli/api"
	"cli/ui"
	ctime "cli/util/time"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"time"
)

type MetricsCommand struct {
	RequiredlArgs MetricsPositionalArgs `positional-args:"yes"`
	StartTime     string                `long:"start" description:"start time of metrics collected with format \"yyyy-MM-ddTHH:mm:ss+/-HH:mm\" or \"yyyy-MM-ddTHH:mm:ssZ\", default to very beginning if not specified."`
	EndTime       string                `long:"end" description:"end time of the metrics collected with format \"yyyy-MM-ddTHH:mm:ss+/-HH:mm\" or \"yyyy-MM-ddTHH:mm:ssZ\", default to current time if not speficied."`
	RecordNumber  string                `long:"number" short:"n" description:"the number of the records to return, will be ignored if both start time and end time are specified."`
	Desc          bool                  `long:"desc" description:"display in descending order, default to ascending order if not specified."`
	Output        string                `long:"output" description:"dump the policy to a file in JSON format"`
}

type MetricsPositionalArgs struct {
	AppName    string `positional-arg-name:"APP_NAME" required:"true"`
	MetricName string `positional-arg-name:"METRIC_NAME" required:"true" description:"available metric supported: \n memoryused, memoryutil, responsetime, throughput, cpu"`
}

func (command MetricsCommand) Execute([]string) error {

	switch command.RequiredlArgs.MetricName {
	case "memoryused":
	case "memoryutil":
	case "responsetime":
	case "throughput":
	case "cpu":
	default:
		return errors.New(fmt.Sprintf(ui.UnrecognizedMetricName, command.RequiredlArgs.MetricName))
	}

	var (
		st     int64 = 0
		et     int64 = time.Now().UnixNano()
		rn     int64 = 0
		err    error
		writer *os.File
	)
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
	if command.RecordNumber != "" {
		rn, err = strconv.ParseInt(command.RecordNumber, 10, 64)
		if rn <= 0 || err != nil {
			return errors.New(fmt.Sprintf(ui.InvalidRecordNumber, command.RecordNumber))
		}
	}
	if command.StartTime != "" && command.EndTime != "" {
		rn = math.MaxInt64
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
	return RetrieveAggregatedMetrics(AutoScaler.CLIConnection,
		command.RequiredlArgs.AppName, command.RequiredlArgs.MetricName,
		st, et, rn, command.Desc, writer, command.Output)
}

func RetrieveAggregatedMetrics(cliConnection api.Connection, appName, metricName string, startTime, endTime, recordNumber int64, desc bool, writer io.Writer, outputfile string) error {

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
		ui.SayMessage(ui.SaveAggregatedMetricHint, appName, outputfile)
	} else {
		ui.SayMessage(ui.ShowAggregatedMetricsHint, metricName, appName)
	}

	table := ui.NewTable(writer, []string{"Metrics Name", "Value", "Timestamp"})
	var (
		page          uint64 = 1
		currentNumber int64  = 0
		next          bool   = true
		noResult      bool   = true
		data          [][]string
	)
	for true {
		next, data, err = apihelper.GetAggregatedMetrics(metricName, startTime, endTime, desc, page)
		if err != nil {
			return err
		}

		for _, row := range data {
			if recordNumber == 0 || currentNumber < recordNumber {
				table.Add(row)
				currentNumber++
			}
		}
		if len(data) > 0 {
			noResult = false
			table.Print()
		}

		if !next || currentNumber >= recordNumber {
			break
		}
		page += 1
	}

	if noResult {
		ui.SayOK()
		ui.SayMessage(ui.AggregatedMetricsNotFound, metricName, appName)

	} else {

		if writer != os.Stdout {
			ui.SayOK()
		}
	}

	return nil
}
