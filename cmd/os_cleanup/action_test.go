package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jjo/openstack-ops/pkg/openstack"
)

func Test_codeNum(t *testing.T) {
	type args struct {
		str    string
		strMap map[string]int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			"codeNum: list",
			args{
				"list",
				actionsMap,
			},
			LIST,
		},
		{
			"codeNum: stop",
			args{
				"delete",
				actionsMap,
			},
			DELETE,
		},
		{
			"codeNum: stop",
			args{
				"delete",
				actionsMap,
			},
			DELETE,
		},
		{
			"codeNum: table",
			args{
				"table",
				outputMap,
			},
			TABLE,
		},
		{
			"codeNum: markdown",
			args{
				"md",
				outputMap,
			},
			MARKDOWN,
		},
	}
	for _, tt := range tests {
		tearDownTest := setupTest(t)
		defer tearDownTest(t)

		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, codeNum(tt.args.str, tt.args.strMap), tt.name)
		})
	}
}

func Test_actionRun(t *testing.T) {
	type args struct {
		osClient   openstack.OSClientInterface
		instances  []openstack.OSResourceInterface
		filter     func(r openstack.OSResourceInterface) bool
		actionCode int
		outputCode int
	}
	tests := []struct {
		name         string
		args         args
		expInstances []openstack.OSResourceInterface
	}{
		{
			"actionRun: list all instances",
			args{
				(&mockOSclient{}).WithProjectToEmail(projectToEmailFunc),
				nil,
				func(r openstack.OSResourceInterface) bool { return true },
				LIST,
				JSON,
			},
			mockInstances,
		},
		{
			"actionRun: list no instance",
			args{
				(&mockOSclient{}).WithProjectToEmail(projectToEmailFunc),
				nil,
				func(r openstack.OSResourceInterface) bool { return false },
				LIST,
				JSON,
			},
			[]openstack.OSResourceInterface{},
		},
		{
			"actionRun: list one instance",
			args{
				(&mockOSclient{}).WithProjectToEmail(projectToEmailFunc),
				nil,
				func(r openstack.OSResourceInterface) bool { return r.(*mockOSResource).ID == "1" },
				LIST,
				JSON,
			},
			[]openstack.OSResourceInterface{mockInstances[0]},
		},
	}

	for _, tt := range tests {
		tearDownTest := setupTest(t)
		defer tearDownTest(t)

		opts := cliOptions{}

		outFile, err := ioutil.TempFile("", "testout")
		if err != nil {
			t.Error(err)
		}
		defer syscall.Unlink(outFile.Name())
		tt.args.instances, err = tt.args.osClient.GetInstances(tt.args.filter)
		if err != nil {
			t.Error(err)
		}
		t.Run(tt.name, func(t *testing.T) {
			actionRun(tt.args.osClient, tt.args.instances, tt.args.actionCode, tt.args.outputCode, outFile, &opts)
			content, _ := os.ReadFile(outFile.Name())
			var result []interface{}
			err := json.Unmarshal(content, &result)
			if err != nil {
				t.Error(err)
			}
			require.Equal(t, len(tt.expInstances), len(result), tt.name)
			for i, m := range tt.expInstances {
				id, name, project := m.GetData()
				r := result[i].(map[string]interface{})
				require.Equal(t, id, r["id"], tt.name)
				require.Equal(t, name, r["name"], tt.name)
				require.Equal(t, project, r["project"], tt.name)
				require.Equal(t, projectToEmailFunc(m), r["email"], tt.name)
			}
		})
	}
}

func Test_actionPerResource(t *testing.T) {
	type args struct {
		resources  []openstack.OSResourceInterface
		actionCode int
	}
	tests := []struct {
		name       string
		args       args
		wantedCall func(*mockOSResource) int
	}{
		{
			"actionPerResource: Delete() calls",
			args{mockInstances, DELETE},
			func(m *mockOSResource) int { return m.calledDelete },
		},
		{
			"actionPerResource: Stop() calls",
			args{mockInstances, STOP},
			func(m *mockOSResource) int { return m.calledStop },
		},
		{
			"actionPerResource: Start() calls",
			args{mockInstances, START},
			func(m *mockOSResource) int { return m.calledStart },
		},
		{
			"actionPerResource: Tag() calls",
			args{mockInstances, TAG},
			func(m *mockOSResource) int { return m.calledTag },
		},
		{
			"actionPerResource: Untag() calls",
			args{mockInstances, UNTAG},
			func(m *mockOSResource) int { return m.calledUntag },
		},
	}
	for _, tt := range tests {
		tearDownTest := setupTest(t)
		defer tearDownTest(t)

		t.Run(tt.name, func(t *testing.T) {
			opts := cliOptions{}
			// Should show function (Delete, Stop, etc) called once
			opts.yes = true
			actionPerResource(tt.args.resources, tt.args.actionCode, &opts)
			for _, m := range tt.args.resources {
				require.Equal(t, 1, tt.wantedCall(m.(*mockOSResource)), tt.name)
			}
			// Should not increase called count (doit=false)
			opts.yes = false
			actionPerResource(tt.args.resources, tt.args.actionCode, &opts)
			for _, m := range tt.args.resources {
				require.Equal(t, 1, tt.wantedCall(m.(*mockOSResource)), tt.name)
			}
		})
	}
}
