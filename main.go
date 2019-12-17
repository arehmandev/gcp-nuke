package main

import (
	"fmt"
	"log"
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
			err := recursiveDeletion(resourceMap, resource, nil)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func recursiveDeletion(resourceMap map[string]gcp.Resource, resource gcp.Resource, recursionList []gcp.Resource) error {

	if sliceContainsResource(recursionList, resource) {
		return fmt.Errorf("resource %v has broken recursion as it is already in the recursion chain: %v", resource.Name(), recursionList)
	}
	recursionList = append(recursionList, resource)

	if len(resource.Dependencies()) > 0 {
		for _, dependency := range resource.Dependencies() {
			recursiveDeletion(resourceMap, resourceMap[dependency], recursionList)
		}
	}
	resource.Remove()

	return nil
}

func sliceContainsResource(inputList []gcp.Resource, input gcp.Resource) bool {
	for _, value := range inputList {
		if value == input {
			return true
		}
	}
	return false
}
