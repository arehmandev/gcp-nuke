package gcp

import (
	"context"
	"log"

	"github.com/arehmandev/gcp-nuke/config"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
)

// ResourceBase -
type ResourceBase struct {
	resourceNames []string
	config        config.Config
}

// Resource -
type Resource interface {
	Name() string
	Setup(config config.Config)
	List() []string
	Dependencies() []string
	Remove() error
}

var resourceMap = make(map[string]Resource)
var ctx = context.Background()

func register(resource Resource) {
	_, exists := resourceMap[resource.Name()]
	if exists {
		log.Fatalf("a resource with the name %s already exists", resource.Name())
	}
	resourceMap[resource.Name()] = resource
}

// GetResourceMap -
func GetResourceMap(config config.Config) map[string]Resource {
	for _, resource := range resourceMap {
		resource.Setup(config)
	}

	return resourceMap
}

// GetZones -
func GetZones(defaultContext context.Context, project string) []string {
	log.Println("[Info] Retrieving zones for project:", project)
	client, err := google.DefaultClient(defaultContext, compute.ComputeScope)
	if err != nil {
		log.Fatal(err)
	}
	serviceClient, err := compute.New(client)
	if err != nil {
		log.Fatal(err)
	}
	zoneListCall := serviceClient.Zones.List(project)
	zoneList, err := zoneListCall.Do()
	if err != nil {
		log.Fatal(err)
	}

	zoneStringSlice := []string{}
	for _, zone := range zoneList.Items {
		zoneStringSlice = append(zoneStringSlice, zone.Name)
	}
	return zoneStringSlice
}
