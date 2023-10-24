package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jjo/openstack-ops/pkg/openstack"

	"github.com/jedib0t/go-pretty/v6/table"
)

const (
	LIST = iota
	STOP
	START
	DELETE
	TAG
	UNTAG
)

const (
	TABLE = iota
	JSON
	CSV
	HTML
	MARKDOWN
)

var (
	actionsMap = map[string]int{
		"list":   LIST,
		"stop":   STOP,
		"start":  START,
		"delete": DELETE,
		"tag":    TAG,
		"untag":  UNTAG,
	}
	outputMap = map[string]int{
		"table": TABLE,
		"json":  JSON,
		"csv":   CSV,
		"html":  HTML,
		"md":    MARKDOWN,
	}
)

func codeNum(str string, strMap map[string]int) int {
	ret, ok := strMap[str]
	if !ok {
		return -1
	}
	return ret
}

func actionRun(osClient *openstack.OSClient, instances []openstack.OSResourceInterface, actionCode, outputCode int) {
	switch actionCode {
	case LIST:
		actionList(instances, outputCode)
	case STOP, START:
		actionStopStart(instances, actionCode)
	case DELETE:
		actionDelete(instances)
	case TAG:
		actionTag(instances)
	case UNTAG:
		actionUnTag(instances)
	}
}

func getTableWriter(instances []openstack.OSResourceInterface) table.Writer {
	tw := table.NewWriter()
	tw.AppendHeader(openstack.GetRowHeader(instances))
	for _, resource := range instances {
		tw.AppendRow(resource.GetRow())
	}

	tw.SortBy([]table.SortBy{{Name: "Project"}, {Name: "Created", Mode: table.Dsc}})
	return tw
}

func actionList(instances []openstack.OSResourceInterface, outputCode int) {
	switch outputCode {
	case TABLE:
		tw := getTableWriter(instances)
		tw.SetStyle(table.StyleLight)
		fmt.Println(tw.Render())
	case CSV:
		tw := getTableWriter(instances)
		fmt.Println(tw.RenderCSV())
	case HTML:
		tw := getTableWriter(instances)
		fmt.Println(tw.RenderHTML())
	case MARKDOWN:
		tw := getTableWriter(instances)
		fmt.Println(tw.RenderMarkdown())
	case JSON:
		// Write the JSON instances to os.Stdout
		jsonData, err := json.MarshalIndent(instances, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		_, err = os.Stdout.Write(jsonData)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func actionDelete(resources []openstack.OSResourceInterface) {
	for _, resource := range resources {
		switch yes {
		case true:
			log.Infof("Deleting server: %s\n", resource.String())
			err := resource.Delete()
			if err != nil {
				log.Errorf("Error deleting: %s: %s\n", resource.String(), err)
			}
		case false:
			log.Infof("**NOT** Deleting (needs --yes): %s\n", resource.String())
		}
	}
}

func actionStopStart(resources []openstack.OSResourceInterface, actionCode int) {
	var err error
	for _, resource := range resources {
		switch actionCode {
		case STOP:
			switch yes {
			case true:
				log.Infof("Stopping: %s\n", resource.String())
				err = resource.Stop()
			case false:
				log.Infof("**NOT** Stopping (needs --yes): %s\n", resource.String())
			}
		case START:
			switch yes {
			case true:
				log.Infof("Starting: %s\n", resource.String())
				err = resource.Start()
			case false:
				log.Infof("**NOT** Starting (needs --yes): %s\n", resource.String())
			}
		}
		if err != nil {
			log.Errorf("Error starting/stopping server %s: %s\n", resource.String(), err)
		}
	}
}

func actionTag(resources []openstack.OSResourceInterface) {
	for _, resource := range resources {
		log.Infof("Tagging: %s\n", resource.String())
		err := resource.Tag(tagValue)
		if err != nil {
			log.Errorf("Error tagging %s\n", resource.String(), err)
		}
	}
}

func actionUnTag(resources []openstack.OSResourceInterface) {
	for _, resource := range resources {
		log.Infof("Untagging: %s\n", resource.String())
		err := resource.Untag(tagValue)
		if err != nil {
			log.Errorf("Error untagging: %s: %s\n", resource.String(), err)
		}
	}
}
