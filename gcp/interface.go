package gcp

import (
	"context"
	"log"

	"github.com/arehmandev/gcp-nuke/config"
)

// ResourceBase -
type ResourceBase struct {
	resourceNames []string
	config        config.Config
	cache         bool
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
