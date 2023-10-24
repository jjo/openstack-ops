package main

import (
	"os"
	"time"

	"github.com/jjo/openstack-ops/pkg/openstack"

	logging "github.com/op/go-logging"
	"github.com/spf13/pflag"
)

const (
	osCleanupTag = "os-cleanup"
	workerCount  = 10
)

var (
	action    string
	output    string
	includeRe string
	excludeRe string
	nDays     int
	tagged    bool
	logLevel  string
	tagValue  string
	yes       bool
	workers   int
	log       *logging.Logger
)

func setupLogging(module string, out *os.File, logLevel string) *logging.Logger {
	log = logging.MustGetLogger(module)
	level, err := logging.LogLevel(logLevel)
	if err != nil {
		log.Fatalf("Invalid log level: %s", logLevel)
	}
	format := logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000}: %{level:.6s} %{id:03x}%{color:reset} %{message}`,
	)
	backend := logging.NewLogBackend(out, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format)
	logging.SetBackend(backendFormatter)
	logging.SetLevel(level, module)
	return log
}

func parseFlags() {
	// Parse the command-line arguments
	pflag.StringVarP(&includeRe, "include-re", "i", "(.+)__(alumno|gmail).*", "regex for instance projects to include")
	pflag.StringVarP(&excludeRe, "exclude-re", "e", "", "regex for instance projects,names,etc to exclude")
	pflag.StringVarP(&action, "action", "a", "", "action to perform: list, stop, start, delete, tag, untag")
	pflag.StringVarP(&output, "output", "o", "table", "output format: table, json, csv, html, md")
	pflag.IntVarP(&nDays, "days", "d", 60, "instances older than `days`")
	pflag.BoolVarP(&tagged, "tagged", "t", false, "list only tagged instances")
	pflag.StringVarP(&tagValue, "tag-value", "", osCleanupTag, "tag value to use")
	pflag.BoolVarP(&yes, "yes", "", false, "commit dangerous actions, e.g. delete")
	pflag.StringVarP(&logLevel, "loglevel", "l", "info", "set log level: debug, info, notice, warning, error, critical")
	pflag.IntVarP(&workers, "workers", "w", workerCount, "number of workers")
	pflag.Parse()
}

func main() {
	parseFlags()

	log = setupLogging("os-cleanup", os.Stderr, logLevel)
	if action == "" {
		log.Fatal("No action specified with: -a <action>, e.g.: -a list")
	}

	actionCode := codeNum(action, actionsMap)
	if actionCode == -1 {
		log.Fatalf("Invalid action: %s", action)
	}
	outputCode := codeNum(output, outputMap)
	if outputCode == -1 {
		log.Fatalf("Invalid output: %s", output)
	}
	// Calculate the timestamp for nDays ago
	nDaysAgo := time.Now().AddDate(0, 0, -nDays)

	osClient := openstack.NewOSClient(log)

	filter := openstack.NewOSResourceFilter(nDaysAgo, includeRe, excludeRe, tagValue, tagged)
	filterFunc := func(resource openstack.OSResourceInterface) bool {
		return filter.Run(resource)
	}

	instances, err := osClient.GetInstances(workers, filterFunc)
	if err != nil {
		log.Fatal("Error while getting instances:", err)
	}

	actionRun(osClient, instances, actionCode, outputCode)
}
