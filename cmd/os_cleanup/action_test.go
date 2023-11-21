package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jjo/openstack-ops/pkg/openstack"
)

func Test_codeNum(t *testing.T) {
	type args struct {
		str    string
		strMap map[string]int
	}

	t.Parallel()

	tests := []struct {
		name string
		args args
		want int
	}{
		{
			"codeNum: list",
			args{"list", actionsMap},
			LIST,
		},
		{
			"codeNum: stop",
			args{"delete", actionsMap},
			DELETE,
		},
		{
			"codeNum: stop",
			args{"stop", actionsMap},
			STOP,
		},
		{
			"codeNum: start",
			args{"start", actionsMap},
			START,
		},
		{
			"codeNum: table",
			args{"table", outputMap},
			TABLE,
		},
		{
			"codeNum: markdown",
			args{"md", outputMap},
			MARKDOWN,
		},
	}

	for _, tt := range tests {
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

	mockInstances := NewMockInstances()
	t.Parallel()

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
			[]openstack.OSResourceInterface{m1},
		},
	}

	for _, tt := range tests {
		opts := cliOptions{}

		outFile, err := os.CreateTemp("", "testout")
		if err != nil {
			t.Error(err)
		}

		defer os.Remove(outFile.Name())

		tt.args.instances, err = tt.args.osClient.GetInstances(tt.args.filter)
		if err != nil {
			t.Error(err)
		}

		t.Run(tt.name, func(t *testing.T) {
			err := actionRun(tt.args.instances, tt.args.actionCode, tt.args.outputCode, outFile, &opts)
			if err != nil {
				t.Error(err)
			}
			content, _ := os.ReadFile(outFile.Name())
			var result []interface{}
			err = json.Unmarshal(content, &result)
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
		actionCode int
	}

	t.Parallel()

	tests := []struct {
		name       string
		args       args
		wantedCall func(*mockOSResource) int
	}{
		{
			"actionPerResource: Delete() calls",
			args{DELETE},
			func(m *mockOSResource) int { return m.calledDelete },
		},
		{
			"actionPerResource: Stop() calls",
			args{STOP},
			func(m *mockOSResource) int { return m.calledStop },
		},
		{
			"actionPerResource: Start() calls",
			args{START},
			func(m *mockOSResource) int { return m.calledStart },
		},
		{
			"actionPerResource: Tag() calls",
			args{TAG},
			func(m *mockOSResource) int { return m.calledTag },
		},
		{
			"actionPerResource: Untag() calls",
			args{UNTAG},
			func(m *mockOSResource) int { return m.calledUntag },
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			resources := NewMockInstances()
			opts := cliOptions{}
			// Should show function (Delete, Stop, etc) called once
			opts.doit = true
			err := actionPerResource(resources, tt.args.actionCode, &opts)
			if err != nil {
				t.Error(err)
			}
			for _, m := range resources {
				require.Equal(t, 1, tt.wantedCall(m.(*mockOSResource)), tt.name)
			}
			// Should not increase called count (doit=false)
			opts.doit = false
			err = actionPerResource(resources, tt.args.actionCode, &opts)
			if err != nil {
				t.Error(err)
			}
			for _, m := range resources {
				require.Equal(t, 1, tt.wantedCall(m.(*mockOSResource)), tt.name)
			}
		})
	}
}
