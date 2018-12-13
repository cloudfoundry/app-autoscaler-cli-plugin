package commands

import (
	"cli/api"
	"cli/ui"
	ctime "cli/util/time"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

type MetricsCommand struct {
	RequiredlArgs MetricsPositionalArgs `positional-args:"yes"`
	InstanceIndex string                `long:"app-instance-index" short:"i" description:"Application instance index, default to empty so aggregated metrics will be returned."`
	StartTime     string                `long:"start" description:"start time of metrics collected with format \"yyyy-MM-ddTHH:mm:ss+/-HH:mm\" or \"yyyy-MM-ddTHH:mm:ssZ\", default to very beginning if not specified."`
	EndTime       string                `long:"end" description:"end time of the metrics collected with format \"yyyy-MM-ddTHH:mm:ss+/-HH:mm\" or \"yyyy-MM-ddTHH:mm:ssZ\", default to current time if not speficied."`
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
		index  int64 = int64(-1)
		st     int64 = 0
		et     int64 = time.Now().UnixNano()
		err    error
		writer *os.File
	)
	if command.InstanceIndex != "" {
		index, err = strconv.ParseInt(command.InstanceIndex, 10, 64)
		if err != nil {
			return errors.New(fmt.Sprintf(ui.InvalidInstanceIndex, command.InstanceIndex))
		}
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

	if command.Output != "" {
		writer, err = os.OpenFile(command.Output, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
		defer writer.Close()
	} else {
		writer = os.Stdout
	}
	if index >= 0 {
		return RetrieveInstanceMetrics(AutoScaler.CLIConnection,
			command.RequiredlArgs.AppName, command.RequiredlArgs.MetricName, index,
			st, et, command.Desc, writer, command.Output)
	} else {
		return RetrieveAggregatedMetrics(AutoScaler.CLIConnection,
			command.RequiredlArgs.AppName, command.RequiredlArgs.MetricName,
			st, et, command.Desc, writer, command.Output)
	}
}

func RetrieveInstanceMetrics(cliConnection api.Connection, appName, metricName string, instanceIndex, startTime, endTime int64, desc bool, writer io.Writer, outputfile string) error {

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

	if outputfile != "" {
		ui.SayMessage(ui.SaveInstanceMetricHint, appName, outputfile)
	} else {
		ui.SayMessage(ui.ShowInstanceMetricsHint, appName)
	}

	table := ui.NewTable(writer, []string{"Metrics Name", "Instance Index", "Value", "At"})
	var (
		page     uint64 = 1
		next     bool   = true
		noResult bool   = true
		data     [][]string
	)
	for next {
		next, data, err = apihelper.GetInstanceMetrics(metricName, instanceIndex, startTime, endTime, desc, page)
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

		if !next {
			break
		}
		page += 1
	}

	if noResult {
		ui.SayOK()
		ui.SayMessage(ui.InstanceMetricsNotFound, appName)
	} else {
		if outputfile != "" {
			ui.SayOK()
		}
	}

	return nil
}

func RetrieveAggregatedMetrics(cliConnection api.Connection, appName, metricName string, startTime, endTime int64, desc bool, writer io.Writer, outputfile string) error {

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

	if outputfile != "" {
		ui.SayMessage(ui.SaveAggregatedMetricHint, appName, outputfile)
	} else {
		ui.SayMessage(ui.ShowAggregatedMetricsHint, appName)
	}

	table := ui.NewTable(writer, []string{"Metrics Name", "Value", "Timestamp"})
	var (
		page     uint64 = 1
		next     bool   = true
		noResult bool   = true
		data     [][]string
	)
	for next {
		next, data, err = apihelper.GetAggregatedMetrics(metricName, startTime, endTime, desc, page)
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

		if !next {
			break
		}
		page += 1
	}

	if noResult {
		ui.SayOK()
		ui.SayMessage(ui.AggregatedMetricsNotFound, appName)
		ui.SayMessage(ui.UseInstanceMetricMsg)

	} else {

		if writer != os.Stdout {
			ui.SayOK()
		}
	}

	return nil
}
