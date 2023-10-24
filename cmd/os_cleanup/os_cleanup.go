package main

import (
	"regexp"
	"time"

	"github.com/jjo/openstack-ops/pkg/logger"
	"github.com/jjo/openstack-ops/pkg/openstack"

	"github.com/spf13/pflag"
)

const (
	osCleanupTag = "os-cleanup"
	workerCount  = 10
)

var mailRe = regexp.MustCompile("(.+)__(.+)_project")

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
)

var log = logger.Log

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

	logger.SetLevel(logLevel)
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
	projectToEmailFunc := func(resource openstack.OSResourceInterface) string {
		return mailRe.ReplaceAllString(resource.GetProjectName(), `$1@$2`)
	}

	osClient := openstack.NewOSClient().
		WithWorkers(workers).
		WithProjectToEmail(projectToEmailFunc)

	filter := openstack.NewOSResourceFilter(nDaysAgo, includeRe, excludeRe, tagValue, tagged)
	filterFunc := func(resource openstack.OSResourceInterface) bool {
		return filter.Run(resource)
	}
	instances, err := osClient.GetInstances(filterFunc)
	if err != nil {
		log.Fatal("Error while getting instances:", err)
	}

	actionRun(osClient, instances, actionCode, outputCode)
}
