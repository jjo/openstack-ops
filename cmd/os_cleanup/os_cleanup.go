package main

import (
	"fmt"
	"os"
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

type cliOptions struct {
	action    string
	output    string
	includeRe string
	excludeRe string
	nDays     int
	tagged    bool
	logLevel  string
	tagValue  string
	doit      bool
	workers   int
}

var log = logger.Log

func parseFlags() cliOptions {
	// Parse the command-line arguments
	includeRe := pflag.StringP("include-re", "i", "(.+)__(alumno|gmail).*", "regex for instance projects to include")
	excludeRe := pflag.StringP("exclude-re", "e", "", "regex for instance projects,names,etc to exclude")
	action := pflag.StringP("action", "a", "", "action to perform: list, stop, start, delete, tag, untag")
	output := pflag.StringP("output", "o", "table", "output format: table, json, csv, html, md")
	nDays := pflag.IntP("days", "d", 60, "instances older than `days`")
	tagged := pflag.BoolP("tagged", "t", false, "list only tagged instances")
	tagValue := pflag.StringP("tag-value", "", osCleanupTag, "tag value to use")
	doit := pflag.BoolP("yes", "", false, "commit dangerous actions, e.g. delete")
	logLevel := pflag.StringP("loglevel", "l", "info", "set log level: debug, info, notice, warning, error, critical")
	workers := pflag.IntP("workers", "w", workerCount, "number of workers")
	pflag.Parse()
	return cliOptions{
		action:    *action,
		output:    *output,
		includeRe: *includeRe,
		excludeRe: *excludeRe,
		nDays:     *nDays,
		tagged:    *tagged,
		logLevel:  *logLevel,
		tagValue:  *tagValue,
		doit:      *doit,
		workers:   *workers,
	}
}

func projectToEmailFunc(resource openstack.OSResourceInterface) string {
	return mailRe.ReplaceAllString(resource.GetProjectName(), `$1@$2`)
}

var osClient openstack.OSClientInterface

func runMain(opts cliOptions, outFile *os.File) error {
	_, err := logger.SetLevel(opts.logLevel)
	if err != nil {
		return err
	}
	if opts.action == "" {
		return fmt.Errorf("No action specified with: -a <action>, e.g.: -a list")
	}

	actionCode := codeNum(opts.action, actionsMap)
	if actionCode == -1 {
		return fmt.Errorf("Invalid action: %s", opts.action)
	}
	outputCode := codeNum(opts.output, outputMap)
	if outputCode == -1 {
		return fmt.Errorf("Invalid output: %s", opts.output)
	}
	// Calculate the timestamp for nDays ago
	nDaysAgo := time.Now().AddDate(0, 0, -opts.nDays)

	// Ease unittesting (by overriding osClient)
	if osClient == nil {
		osClient = openstack.NewOSClient().
			WithWorkers(opts.workers).
			WithProjectToEmail(projectToEmailFunc)
	}

	filter := openstack.NewOSResourceFilter(nDaysAgo, opts.includeRe, opts.excludeRe, opts.tagValue, opts.tagged)
	filterFunc := func(resource openstack.OSResourceInterface) bool {
		return filter.Run(resource)
	}
	instances, err := osClient.GetInstances(filterFunc)
	if err != nil {
		log.Error("Error while getting instances:", err)
	}

	return actionRun(osClient, instances, actionCode, outputCode, outFile, &opts)
}
func main() {
	cliOptions := parseFlags()
	err := runMain(cliOptions, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}
