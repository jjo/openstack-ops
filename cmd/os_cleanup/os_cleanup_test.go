package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/jjo/openstack-ops/pkg/openstack"
	"github.com/stretchr/testify/require"
)

type mockOSclient struct {
	projectToEmail func(openstack.OSResourceInterface) string
}

func (m *mockOSclient) WithProjectToEmail(f func(r openstack.OSResourceInterface) string) openstack.OSClientInterface {
	m.projectToEmail = f
	return m
}

func (m *mockOSclient) WithWorkers(int) openstack.OSClientInterface {
	return m
}

type mockOSResource struct {
	osClient     *mockOSclient
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Project      string    `json:"project"`
	Email        string    `json:"email"`
	Created      time.Time `json:"created"`
	Tags         []string  `json:"tags"`
	calledStart  int
	calledStop   int
	calledDelete int
	calledTag    int
	calledUntag  int
}

func (m *mockOSResource) GetData() (string, string, string) {
	return m.ID, m.Name, m.Project
}

func (m *mockOSResource) CreatedBefore(t time.Time) bool {
	return m.Created.Before(t)
}

func (m *mockOSResource) Delete() error {
	m.calledDelete++
	return nil
}

func (m *mockOSResource) GetProjectName() string {
	return m.Project
}

func (m *mockOSResource) GetTags() []string {
	return m.Tags
}

func (m *mockOSResource) Start() error {
	m.calledStart++
	return nil
}

func (m *mockOSResource) Tag(s string) error {
	m.calledTag++
	return nil
}

func (m *mockOSResource) Untag(s string) error {
	m.calledUntag++
	return nil
}

func (m *mockOSResource) Stop() error {
	m.calledStop++
	return nil
}

func (m *mockOSResource) String() string {
	return fmt.Sprintf("%v", *m)
}

func (m *mockOSResource) StringAll() string {
	return fmt.Sprintf("%v", *m)
}

func (m *mockOSResource) GetRow() []interface{} {
	return []interface{}{m.Name, m.ID, m.Created, "active", "RUNNING", m.Project, m.Email, m.GetTags()}
}

func newMockOSResource(id, name, project string, nDaysAgo int, tags []string) *mockOSResource {
	return &mockOSResource{
		ID:      id,
		Name:    name,
		Project: project,
		Created: time.Now().AddDate(0, 0, -nDaysAgo),
	}
}

var (
	nDays1        = 30
	nDays2        = 60
	mockInstances = []openstack.OSResourceInterface{
		newMockOSResource("1", "one", "foo__bar.com_project", nDays1, []string{"tag1"}),
		newMockOSResource("2", "two", "foo__bar.com_project", nDays2, []string{"tag2"}),
	}
)

func (m *mockOSclient) GetInstances(filter func(r openstack.OSResourceInterface) bool) ([]openstack.OSResourceInterface, error) {
	instances := make([]openstack.OSResourceInterface, 0)
	for _, i := range mockInstances {
		instance := i.(*mockOSResource)
		instance.osClient = m
		if m.projectToEmail != nil {
			instance.Email = m.projectToEmail(instance)
		}
		if filter(instance) {
			instances = append(instances, instance)
		}
	}
	return instances, nil
}

func setupTest(t *testing.T) func(t *testing.T) {
	osClient = &mockOSclient{}
	return func(t *testing.T) {
	}
}

func Test_runMain(t *testing.T) {
	type args struct {
		opts cliOptions
	}
	tests := []struct {
		name       string
		args       args
		wantLength int
		wantErr    bool
	}{
		{
			"runMain: bad logLevel",
			args{
				cliOptions{
					action:   "list",
					output:   "json",
					logLevel: "foobar",
				},
			},
			0,
			true,
		},
		{
			"runMain: bad action",
			args{
				cliOptions{
					action:   "listfoo",
					output:   "json",
					logLevel: "debug",
				},
			},
			0,
			true,
		},
		{
			"runMain: bad output",
			args{
				cliOptions{
					action:   "list",
					output:   "foobar",
					logLevel: "debug",
				},
			},
			0,
			true,
		},
		{
			"runMain: list all from 0 days ago",
			args{
				cliOptions{
					action:    "list",
					output:    "json",
					includeRe: "(.+)__.*",
					excludeRe: "",
					nDays:     0,
					tagValue:  "",
					tagged:    false,
					logLevel:  "info",
					yes:       false,
					workers:   10,
				},
			},
			len(mockInstances),
			false,
		},
		{
			"runMain: list all from just after newest",
			args{
				cliOptions{
					action:    "list",
					output:    "json",
					includeRe: "(.+)__.*",
					excludeRe: "",
					nDays:     nDays1 - 1,
					tagValue:  "",
					tagged:    false,
					logLevel:  "info",
					yes:       false,
					workers:   10,
				},
			},
			len(mockInstances),
			false,
		},
		{
			"runMain: list oldest",
			args{
				cliOptions{
					action:    "list",
					output:    "json",
					includeRe: "(.+)__.*",
					excludeRe: "",
					nDays:     nDays1 + 1,
					tagValue:  "",
					tagged:    false,
					logLevel:  "info",
					yes:       false,
					workers:   10,
				},
			},
			1,
			false,
		},
	}
	osClient = &mockOSclient{}
	for _, tt := range tests {
		tearDownTest := setupTest(t)
		defer tearDownTest(t)

		outFile, err := ioutil.TempFile("", "testout")
		if err != nil {
			t.Error(err)
		}
		defer syscall.Unlink(outFile.Name())
		t.Run(tt.name, func(t *testing.T) {
			err := runMain(tt.args.opts, outFile)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			content, _ := os.ReadFile(outFile.Name())
			var result []interface{}
			err = json.Unmarshal(content, &result)
			if err != nil {
				t.Error(err)
			}
			require.Equal(t, tt.wantLength, len(result), tt.name)
		})
	}
}
