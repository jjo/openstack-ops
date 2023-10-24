package openstack

import (
	"sync"

	"github.com/alitto/pond"

	"github.com/jjo/openstack-ops/pkg/logger"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/tags"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	"github.com/gophercloud/gophercloud/pagination"
)

type OSClientInterface interface {
	GetInstances(filter func(OSResourceInterface) bool) ([]OSResourceInterface, error)
	WithWorkers(workers int) OSClientInterface
	WithProjectToEmail(projectToEmail func(OSResourceInterface) string) OSClientInterface
}

type OSClient struct {
	ProviderClient *gophercloud.ProviderClient
	ComputeClient  *gophercloud.ServiceClient
	IdentityClient *gophercloud.ServiceClient
	workers        int
	projectToEmail func(OSResourceInterface) string
	projectsCache  map[string]string
}

var log = logger.Log

func NewOSClient() *OSClient {
	// Load your OpenStack credentials from environment variables or configuration file
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

	log.Debugf("Successfully created clients for auth_url=%s domain=%s user=%s project=%s",
		authOpts.IdentityEndpoint, authOpts.DomainName, authOpts.Username, authOpts.TenantName)
	return &OSClient{
		ProviderClient: provider,
		ComputeClient:  computeClient,
		IdentityClient: identityClient,
		workers:        1,
	}
}

func (osClient *OSClient) withProjectsCache() (*OSClient, error) {
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
	log.Debugf("Created projectsCache: Loaded %d projects", len(osClient.projectsCache))

	return osClient, nil
}

func (osClient *OSClient) WithWorkers(workers int) OSClientInterface {
	log.Debugf("Setting workers to: %d", workers)
	osClient.workers = workers
	return osClient
}

func (osClient *OSClient) WithProjectToEmail(projectToEmail func(OSResourceInterface) string) OSClientInterface {
	log.Debugf("Setting projectToEmail to: %s", projectToEmail)
	osClient.projectToEmail = projectToEmail
	return osClient
}

func (osClient *OSClient) GetInstances(
	filter func(OSResourceInterface) bool,
) ([]OSResourceInterface, error) {
	osClient, err := osClient.withProjectsCache()
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
	pool := pond.New(osClient.workers, 0, pond.MinWorkers(osClient.workers))
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
			instance := Instance{
				osClient:     osClient,
				Server:       &server.Server,
				InstanceName: server.Server.Name,
				InstanceID:   server.Server.ID,
				Created:      server.Server.Created,
				VmState:      server.VmState,
				PowerState:   server.PowerState.String(),
				ProjectName:  projectName,
				Tags:         serverTags,
			}
			if osClient.projectToEmail != nil {
				instance.Email = osClient.projectToEmail(&instance)
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
