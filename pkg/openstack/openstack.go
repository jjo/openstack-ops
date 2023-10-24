package openstack

import (
	"regexp"
	"sync"

	"github.com/alitto/pond"
	"github.com/op/go-logging"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/tags"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	"github.com/gophercloud/gophercloud/pagination"
)

var mailRe = regexp.MustCompile("(.+)__(.+)_project")

type OSClient struct {
	ProviderClient *gophercloud.ProviderClient
	ComputeClient  *gophercloud.ServiceClient
	IdentityClient *gophercloud.ServiceClient
	projectsCache  map[string]string
}

var log *logging.Logger

func NewOSClient(globalLog *logging.Logger) *OSClient {
	// Load your OpenStack credentials from environment variables or configuration file
	log = globalLog
	authOpts, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		log.Fatalf("Failed to get OpenStack authentication options (missing OS_* env vars):", err)
	}

	// Create an authenticated OpenStack client
	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		log.Fatal("Failed to create an authenticated client:", err)
	}

	// Initialize the Compute service client
	computeClient, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{})

	// NB: v2.26 needed to enable Tags interface
	computeClient.Microversion = "2.26"
	if err != nil {
		log.Fatal("Failed to create Compute service client:", err)
	}
	// Initialize the identity service client
	identityClient, err := openstack.NewIdentityV3(provider, gophercloud.EndpointOpts{})
	if err != nil {
		log.Fatal("Failed to create Identity service client:", err)
	}

	return &OSClient{
		ProviderClient: provider,
		ComputeClient:  computeClient,
		IdentityClient: identityClient,
	}
}

func WithProjectsCache(osClient *OSClient) (*OSClient, error) {
	if osClient.projectsCache != nil {
		return osClient, nil
	}
	osClient.projectsCache = make(map[string]string)
	projectPager := projects.List(osClient.IdentityClient, projects.ListOpts{})
	// Retrieve and store project information
	err := projectPager.EachPage(func(page pagination.Page) (bool, error) {
		projectList, err := projects.ExtractProjects(page)
		if err != nil {
			return false, err
		}
		for _, project := range projectList {
			osClient.projectsCache[project.ID] = project.Name
		}
		return true, nil
	})
	if err != nil {
		log.Fatalf("Failed to paginate projects: %s", err)
		return nil, err
	}

	return osClient, nil
}

func (osClient *OSClient) GetInstances(workers int, filter func(OSResourceInterface) bool) ([]OSResourceInterface, error) {
	osClient, err := WithProjectsCache(osClient)
	if err != nil {
		return nil, err
	}

	if err != nil {
		log.Fatalf("Failed to paginate projects: %s", err)
		return nil, err
	}

	var allServers []ServerWithExt

	allPages, err := servers.List(osClient.ComputeClient, servers.ListOpts{
		AllTenants: true,
	}).AllPages()
	if err != nil {
		log.Fatalf("Failed to list servers: %s", err)
		return nil, err
	}

	err = servers.ExtractServersInto(allPages, &allServers)
	if err != nil {
		log.Fatalf("Failed to extract servers: %s", err)
		return nil, err
	}

	instances := make([]OSResourceInterface, 0)
	// Iterate over the paginated results and filter instances older than one month
	mu := &sync.Mutex{}
	pool := pond.New(workers, 0, pond.MinWorkers(workers))
	for _, serverIterator := range allServers {
		server := new(ServerWithExt)
		*server = serverIterator
		pool.Submit(func() {
			projectName := osClient.projectsCache[server.Server.TenantID]
			resp := tags.List(osClient.ComputeClient, server.Server.ID)
			serverTags, errTmp := resp.Extract()
			if errTmp != nil && resp.StatusCode != 404 {
				log.Errorf("Getting tags for %s: %s", server.Server.ID, errTmp)
				err = errTmp
			}
			mail := mailRe.ReplaceAllString(projectName, `$1@$2`)
			instance := Instance{
				osClient:     osClient,
				Server:       &server.Server,
				InstanceName: server.Server.Name,
				InstanceID:   server.Server.ID,
				Created:      server.Server.Created,
				VmState:      server.VmState,
				PowerState:   server.PowerState.String(),
				ProjectName:  projectName,
				Email:        mail,
				Tags:         serverTags,
			}
			if filter(&instance) {
				mu.Lock()
				instances = append(instances, &instance)
				mu.Unlock()
			}
		})
	}
	pool.StopAndWait()
	return instances, err
}
