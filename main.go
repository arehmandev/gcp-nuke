package main

import (
	"os"

	"github.com/arehmandev/gcp-nuke/config"
	"github.com/arehmandev/gcp-nuke/gcp"
)

func main() {

	config := config.Config{Project: os.Getenv("GCP_PROJECT_ID")}

	// Behaviour to delete one project at a time - will be made into loop later
	removeProject(config)
}

func removeProject(config config.Config) {
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
			recursiveDeletion(resourceMap, resourceMap[dependency])
		}
	}
	resource.Remove()

}
