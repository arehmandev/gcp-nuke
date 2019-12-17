package main

import (
	"os"

	"github.com/arehmandev/gcp-nuke/config"
	"github.com/arehmandev/gcp-nuke/gcp"
)

func main() {

	config := config.Config{Projects: []string{os.Getenv("GCP_PROJECT_ID")}}
	resourceMap := gcp.GetResourceMap(config)

	for _, resource := range resourceMap {
		if len(resource.List()) > 0 {
			recursiveDeletion(resourceMap, resource)
		}
	}
}

func recursiveDeletion(resourceMap map[string]gcp.Resource, resource gcp.Resource) {

	if len(resource.Dependencies()) > 0 {
		for _, dependency := range resource.Dependencies() {
			dependencyResource := resourceMap[dependency]
			if len(dependencyResource.List()) > 0 {
				recursiveDeletion(resourceMap, dependencyResource)
			} else {
				dependencyResource.Remove()
			}
		}
	}
	resource.Remove()
}
