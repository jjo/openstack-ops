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
	case STOP, START, DELETE, TAG, UNTAG:
		actionPerResource(instances, actionCode)
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

func yesnoStr(yes bool, msg string) string {
	yn := map[bool]string{true: "", false: "**NOT**(missing --yes) "}[yes]
	return fmt.Sprintf("%s%s", yn, msg)
}

func actionPerResource(resources []openstack.OSResourceInterface, actionCode int) {
	var err error
	var msg string
	for _, resource := range resources {
		switch actionCode {
		case STOP:
			msg = "Stopping server"
			log.Infof("%s: %s\n", yesnoStr(yes, msg), resource.String())
			if yes {
				err = resource.Stop()
			}
		case START:
			msg = "Starting server"
			log.Infof("%s: %s\n", yesnoStr(yes, msg), resource.String())
			if yes {
				err = resource.Start()
			}
		case DELETE:
			msg = "Deleting server"
			log.Infof("%s: %s\n", yesnoStr(yes, msg), resource.String())
			if yes {
				err = resource.Delete()
			}
		case TAG:
			msg = "Tagging server"
			log.Infof("%s: %s <- %s\n", yesnoStr(yes, msg), resource.String(), tagValue)
			if yes {
				err = resource.Tag(tagValue)
			}
		case UNTAG:
			msg = "Untagging server"
			log.Infof("%s: %s <- %s\n", yesnoStr(yes, msg), resource.String(), tagValue)
			if yes {
				err = resource.Untag(tagValue)
			}
		}
		if err != nil {
			log.Errorf("Error %s %s: %s\n", msg, resource.String(), err)
		}
	}
}
