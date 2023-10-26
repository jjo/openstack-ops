package openstack

import (
	"fmt"
	"time"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/extendedstatus"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/startstop"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/tags"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
)

type OSResourceInterface interface {
	GetData() (string, string, string)
	Stop() error
	Start() error
	Delete() error
	GetTags() []string
	Tag(string) error
	Untag(string) error
	String() string
	StringAll() string
	GetProjectName() string
	CreatedBefore(time.Time) bool
	GetRow() []interface{}
}

type Instance struct {
	osClient     *OSClient
	Server       *servers.Server
	InstanceName string    `json:"name"`
	InstanceID   string    `json:"id"`
	Created      time.Time `json:"created"`
	ProjectName  string    `json:"project"`
	Email        string    `json:"email"`
	VMState      string    `json:"vmstate"`
	PowerState   string    `json:"powerstate"`
	Tags         []string  `json:"tags"`
}

type ServerWithExt struct {
	servers.Server
	extendedstatus.ServerExtendedStatusExt
}

func GetRowHeader([]OSResourceInterface) []interface{} {
	return []interface{}{"Instance_Name", "Instance_ID", "Created", "VMState", "PowerState", "Project", "Email", "Tags"}
}

func (instance *Instance) GetData() (string, string, string) {
	return instance.InstanceID, instance.InstanceName, instance.ProjectName
}

func (instance *Instance) GetRow() []interface{} {
	return []interface{}{
		instance.InstanceName,
		instance.InstanceID,
		instance.Created,
		instance.VMState,
		instance.PowerState,
		instance.ProjectName,
		instance.Email,
		instance.Tags,
	}
}

func (instance *Instance) Delete() error {
	return servers.Delete(instance.osClient.ComputeClient, instance.InstanceID).ExtractErr()
}

func (instance *Instance) Stop() error {
	return startstop.Stop(instance.osClient.ComputeClient, instance.InstanceID).ExtractErr()
}

func (instance *Instance) Start() error {
	return startstop.Start(instance.osClient.ComputeClient, instance.InstanceID).ExtractErr()
}

func (instance *Instance) Tag(str string) error {
	return tags.Add(instance.osClient.ComputeClient, instance.InstanceID, str).ExtractErr()
}

func (instance *Instance) Untag(str string) error {
	resp := tags.Delete(instance.osClient.ComputeClient, instance.InstanceID, str)

	err := resp.ExtractErr()
	if err != nil && resp.StatusCode == 404 {
		err = nil
	}
	return err
}

func (instance *Instance) CreatedBefore(t time.Time) bool {
	return instance.Server.Created.Before(t)
}

func (instance *Instance) String() string {
	return fmt.Sprintf("Kind: Server Name: %s ID: %s Project: %s",
		instance.InstanceName, instance.InstanceID, instance.ProjectName)
}

func (instance *Instance) StringAll() string {
	return fmt.Sprintf("%v", instance)
}

func (instance *Instance) GetTags() []string {
	return instance.Tags
}

func (instance *Instance) GetProjectName() string {
	return instance.ProjectName
}
