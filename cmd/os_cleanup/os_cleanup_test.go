package main

import (
	"encoding/json"
	"fmt"
	"os"
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

func (m *mockOSResource) Tag(_ string) error {
	m.calledTag++
	return nil
}

func (m *mockOSResource) Untag(_ string) error {
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
		Tags:    tags,
		Created: time.Now().AddDate(0, 0, -nDaysAgo),
	}
}
func copyMockOSResource(m *mockOSResource) *mockOSResource {
	return &mockOSResource{
		ID:      m.ID,
		Name:    m.Name,
		Project: m.Project,
		Tags:    m.Tags,
		Created: m.Created,
	}
}

var (
	nDays1 = 30
	nDays2 = 60
	m1     = newMockOSResource("1", "one", "foo__bar.com_project", nDays1, []string{"tag1"})
	m2     = newMockOSResource("2", "two", "foo__bar.com_project", nDays2, []string{"tag2"})
)

func NewMockOSClient() openstack.OSClientInterface {
	return &mockOSclient{}
}
func NewMockInstances() []openstack.OSResourceInterface {
	instances := []openstack.OSResourceInterface{
		copyMockOSResource(m1),
		copyMockOSResource(m2),
	}
	return instances
}

func (m *mockOSclient) GetInstances(
	filter func(r openstack.OSResourceInterface) bool) (
	[]openstack.OSResourceInterface, error,
) {
	instances := make([]openstack.OSResourceInterface, 0)
	mockInstances := NewMockInstances()

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

func Test_runServerMain(t *testing.T) {
	type args struct {
		opts cliOptions
	}

	mockInstances := NewMockInstances()

	tests := []struct {
		name          string
		args          args
		wantInstances []openstack.OSResourceInterface
		wantErr       bool
	}{
		{
			"runServerMain: bad logLevel",
			args{
				cliOptions{
					action:   "list",
					output:   "json",
					logLevel: "foobar",
				},
			},
			[]openstack.OSResourceInterface{},
			true,
		},
		{
			"runServerMain: bad action",
			args{
				cliOptions{
					action:   "listfoo",
					output:   "json",
					logLevel: "debug",
				},
			},
			[]openstack.OSResourceInterface{},
			true,
		},
		{
			"runServerMain: bad output",
			args{
				cliOptions{
					action:   "list",
					output:   "foobar",
					logLevel: "debug",
				},
			},
			[]openstack.OSResourceInterface{},
			true,
		},
		{
			"runServerMain: list all from 0 days ago",
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
					doit:      false,
					workers:   10,
				},
			},
			mockInstances,
			false,
		},
		{
			"runServerMain: list all from just after newest",
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
					doit:      false,
					workers:   10,
				},
			},
			mockInstances,
			false,
		},
		{
			"runServerMain: list oldest",
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
					doit:      false,
					workers:   10,
				},
			},
			[]openstack.OSResourceInterface{m2},
			false,
		},
		{
			"runServerMain: list tagged (one instance)",
			args{
				cliOptions{
					action:    "list",
					output:    "json",
					includeRe: "(.+)__.*",
					excludeRe: "",
					nDays:     0,
					tagValue:  "tag1",
					tagged:    true,
					logLevel:  "info",
					doit:      false,
					workers:   10,
				},
			},
			[]openstack.OSResourceInterface{m1},
			false,
		},
		{
			"runServerMain: list tagged (no instance)",
			args{
				cliOptions{
					action:    "list",
					output:    "json",
					includeRe: "(.+)__.*",
					excludeRe: "",
					nDays:     0,
					tagValue:  "tagFooBar",
					tagged:    true,
					logLevel:  "info",
					doit:      false,
					workers:   10,
				},
			},
			[]openstack.OSResourceInterface{},
			false,
		},
		{
			"runServerMain: list includeRe (one instance)",
			args{
				cliOptions{
					action:    "list",
					output:    "json",
					includeRe: "one",
					excludeRe: "",
					nDays:     0,
					tagValue:  "",
					tagged:    false,
					logLevel:  "info",
					doit:      false,
					workers:   10,
				},
			},
			[]openstack.OSResourceInterface{m1},
			false,
		},
		{
			"runServerMain: list includeRe (one instance)",
			args{
				cliOptions{
					action:    "list",
					output:    "json",
					includeRe: "foo__bar",
					excludeRe: "two",
					nDays:     0,
					tagValue:  "",
					tagged:    false,
					logLevel:  "info",
					doit:      false,
					workers:   10,
				},
			},
			[]openstack.OSResourceInterface{m1},
			false,
		},
		{
			"runServerMain: list exclude (no instance)",
			args{
				cliOptions{
					action:    "list",
					output:    "json",
					includeRe: "FOOone",
					excludeRe: "",
					nDays:     0,
					tagValue:  "",
					tagged:    false,
					logLevel:  "info",
					doit:      false,
					workers:   10,
				},
			},
			[]openstack.OSResourceInterface{},
			false,
		},
	}

	for _, tt := range tests {

		outFile, err := os.CreateTemp("", "testout")
		if err != nil {
			t.Error(err)
		}

		defer os.Remove(outFile.Name())

		osClient := NewMockOSClient()

		t.Run(tt.name, func(t *testing.T) {
			err := runServerMain(osClient, tt.args.opts, outFile)
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
			wantJSON, _ := json.Marshal(tt.wantInstances)
			require.JSONEq(
				t,
				string(wantJSON),
				string(content),
				tt.name,
			)
		})
	}
}
