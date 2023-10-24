package openstack

import (
	"testing"
	"time"
)

// MockOSClient is a mock implementation of OSClient for testing purposes
type MockOSClient struct {
	GetInstances func(func(Instance) bool) ([]Instance, error)
}

type MockInstance struct {
	Created time.Time
}

type MockInstanceInterface interface {
	CreatedBefore(time.Time) bool
}

// You can define mock implementations of OSClient methods here for testing

func TestCreatedBefore(t *testing.T) {
	instance := &Instance{
		Created: time.Now().Add(-24 * time.Hour), // Set the created time to one day ago
	}

	if !instance.CreatedBefore(time.Now()) {
		t.Error("Expected CreatedBefore to return true")
	}
}

func TestInstanceString(t *testing.T) {
	instance := &Instance{
		InstanceName: "TestInstance",
		InstanceID:   "12345",
		TenantName:   "TestTenant",
	}

	expected := "Name: TestInstance ID: 12345 Tenant: TestTenant"
	result := instance.String()

	if result != expected {
		t.Errorf("Expected: %s, Got: %s", expected, result)
	}
}

// You can write similar test functions for other methods of the Instance struct.

func TestGetRowHeader(t *testing.T) {
	// Assuming you have a list of instances to pass as an argument.
	instances := []InstanceInterface{
		&Instance{},
		&Instance{},
		&Instance{},
	}

	expected := []interface{}{"Instance_Name", "Instance_ID", "Created", "VmState", "PowerState", "Project", "Email", "Tags"}
	result := GetRowHeader(instances)

	if len(result) != len(expected) {
		t.Errorf("Expected: %v, Got: %v", expected, result)
		return
	}

	for i, val := range result {
		if val != expected[i] {
			t.Errorf("Expected: %v, Got: %v", expected[i], val)
		}
	}
}

// You can write similar test functions for other methods of the main package.

func TestMockOSClient_GetInstances(t *testing.T) {
	mockClient := &MockOSClient{}

	// Test the GetInstances function with your mock client
	_, err := mockClient.GetInstances(func(Instance) bool { return true })
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
}

// You can add more test functions as needed to cover other parts of your code.

func TestMain(m *testing.M) {
	// This is the entry point for running tests.
	// You can add setup code here if needed.
	// For example, initializing mock clients, setting up test data, etc.
	// Then run the tests with m.Run()
	m.Run()
}
