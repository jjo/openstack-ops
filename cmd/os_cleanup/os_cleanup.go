package main

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/jjo/openstack-ops/pkg/logger"
	"github.com/jjo/openstack-ops/pkg/openstack"

	"github.com/spf13/cobra"
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
var c cliOptions

func projectToEmailFunc(resource openstack.OSResourceInterface) string {
	return mailRe.ReplaceAllString(resource.GetProjectName(), `$1@$2`)
}

var osClient openstack.OSClientInterface

func runServerMain(opts cliOptions, outFile *os.File) error {
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

	return actionRun(instances, actionCode, outputCode, outFile, &opts)
}

func cmdServer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Cleanup unused openstack `server` resources (VM intances)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServerMain(c, os.Stdout)
		},
	}
	pflags := cmd.PersistentFlags()

	pflags.StringVarP(&c.includeRe, "include-re", "i", "(.+)__(alumno|gmail).*", "regex for instance projects to include")
	pflags.StringVarP(&c.excludeRe, "exclude-re", "e", "", "regex for instance projects,names,etc to exclude")

	pflags.StringVarP(&c.action, "action", "a", "", "action to perform: list, stop, start, delete, tag, untag")
	cmd.MarkPersistentFlagRequired("action")

	pflags.StringVarP(&c.output, "output", "o", "table", "output format: table, json, csv, html, md")
	pflags.IntVarP(&c.nDays, "days", "d", 60, "instances older than `days`")

	pflags.BoolVarP(&c.tagged, "tagged", "t", false, "list only tagged instances")
	pflags.StringVarP(&c.tagValue, "tag-value", "", osCleanupTag, "tag value to use")
	pflags.BoolVarP(&c.tagged, "yes", "", false, "commit dangerous actions, e.g. delete")

	pflags.StringVarP(&c.logLevel, "loglevel", "l", "info", "set log level: debug, info, notice, warning, error, critical")
	pflags.IntVarP(&c.workers, "workers", "w", workerCount, "number of workers")
	return cmd
}
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "os_cleanup",
		Short: "Cleanup unused openstack resources",
	}
	rootCmd.AddCommand(cmdServer())
	return rootCmd
}
func main() {
	cmd := NewRootCommand()
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
