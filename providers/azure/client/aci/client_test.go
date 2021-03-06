package aci

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/virtual-kubelet/virtual-kubelet/providers/azure/client/resourcegroups"
	"github.com/google/uuid"
)

var (
	client         *Client
	location       = "eastus"
	resourceGroup  = "virtual-kubelet-tests"
	containerGroup = "virtual-kubelet-test-container-group"
)

func init() {
	// Check if the AZURE_AUTH_LOCATION variable is already set.
	// If it is not set, set it to the root of this project in a credentials.json file.
	if os.Getenv("AZURE_AUTH_LOCATION") == "" {
		// Check if the credentials.json file exists in the root of this project.
		_, filename, _, _ := runtime.Caller(0)
		dir := filepath.Dir(filename)
		file := filepath.Join(dir, "../../../../credentials.json")

		// Check if the file exists.
		if _, err := os.Stat(file); os.IsNotExist(err) {
			log.Fatalf("Either set AZURE_AUTH_LOCATION or add a credentials.json file to the root of this project.")
		}

		// Set the environment variable for the authentication file.
		os.Setenv("AZURE_AUTH_LOCATION", file)
	}

	// Create a resource group name with uuid.
	uid := uuid.New()
	resourceGroup += "-" + uid.String()[0:6]
}

// The TestMain function creates a resource group for testing
// and deletes in when it's done.
func TestMain(m *testing.M) {
	// Check if the resource group exists and create it if not.
	rgCli, err := resourcegroups.NewClient()
	if err != nil {
		log.Fatalf("creating new resourcegroups client failed: %v", err)
	}

	// Check if the resource group exists.
	exists, err := rgCli.ResourceGroupExists(resourceGroup)
	if err != nil {
		log.Fatalf("checking if resource group exists failed: %v", err)
	}

	if !exists {
		// Create the resource group.
		_, err := rgCli.CreateResourceGroup(resourceGroup, resourcegroups.Group{
			Location: location,
		})
		if err != nil {
			log.Fatalf("creating resource group failed: %v", err)
		}
	}

	// Run the tests.
	merr := m.Run()

	// Delete the resource group.
	if err := rgCli.DeleteResourceGroup(resourceGroup); err != nil {
		log.Printf("Couldn't delete resource group %q: %v", resourceGroup, err)

	}

	if merr != 0 {
		os.Exit(merr)
	}

	os.Exit(0)
}

func TestNewClient(t *testing.T) {
	var err error
	client, err = NewClient()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateContainerGroupFails(t *testing.T) {
	_, err := client.CreateContainerGroup(resourceGroup, containerGroup, ContainerGroup{
		Location: location,
		ContainerGroupProperties: ContainerGroupProperties{
			OsType: Linux,
			Containers: []Container{
				{
					Name: "nginx",
					ContainerProperties: ContainerProperties{
						Image:   "nginx",
						Command: []string{"nginx", "-g", "daemon off;"},
						Ports: []ContainerPort{
							{
								Protocol: ContainerNetworkProtocolTCP,
								Port:     80,
							},
						},
					},
				},
			},
		},
	})
	if err == nil {
		t.Fatal("expected create container group to fail with ResourceSomeRequestsNotSpecified, but returned nil")
	}

	if !strings.Contains(err.Error(), "ResourceSomeRequestsNotSpecified") {
		t.Fatalf("expected ResourceSomeRequestsNotSpecified to be in the error message but got: %v", err)
	}
}

func TestCreateContainerGroup(t *testing.T) {
	cg, err := client.CreateContainerGroup(resourceGroup, containerGroup, ContainerGroup{
		Location: location,
		ContainerGroupProperties: ContainerGroupProperties{
			OsType: Linux,
			Containers: []Container{
				{
					Name: "nginx",
					ContainerProperties: ContainerProperties{
						Image:   "nginx",
						Command: []string{"nginx", "-g", "daemon off;"},
						Ports: []ContainerPort{
							{
								Protocol: ContainerNetworkProtocolTCP,
								Port:     80,
							},
						},
						Resources: ResourceRequirements{
							Requests: ResourceRequests{
								CPU:        1,
								MemoryInGB: 1,
							},
							Limits: ResourceLimits{
								CPU:        1,
								MemoryInGB: 1,
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if cg.Name != containerGroup {
		t.Fatalf("resource group name is %s, expected %s", cg.Name, containerGroup)
	}
}

func TestGetContainerGroup(t *testing.T) {
	cg, err, _ := client.GetContainerGroup(resourceGroup, containerGroup)
	if err != nil {
		t.Fatal(err)
	}
	if cg.Name != containerGroup {
		t.Fatalf("resource group name is %s, expected %s", cg.Name, containerGroup)
	}
}

func TestListContainerGroup(t *testing.T) {
	list, err := client.ListContainerGroups(resourceGroup)
	if err != nil {
		t.Fatal(err)
	}
	for _, cg := range list.Value {
		if cg.Name != containerGroup {
			t.Fatalf("resource group name is %s, expected %s", cg.Name, containerGroup)
		}
	}
}

func TestDeleteContainerGroup(t *testing.T) {
	err := client.DeleteContainerGroup(resourceGroup, containerGroup)
	if err != nil {
		t.Fatal(err)
	}
}
