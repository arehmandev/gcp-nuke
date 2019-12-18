package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/arehmandev/gcp-nuke/config"
	"github.com/arehmandev/gcp-nuke/gcp"
	"golang.org/x/sync/errgroup"
)

func main() {

	config := config.Config{
		Project: os.Getenv("GCP_PROJECT_ID"),
		Timeout: 90,
		Context: context.Background(),
	}

	// Behaviour to delete one project at a time - will be made into loop later
	removeProject(config)
}

func removeProject(config config.Config) {
	resourceMap := gcp.GetResourceMap(config)

	// Parallel deletion
	errs, _ := errgroup.WithContext(config.Context)

	for _, resource := range resourceMap {
		resource := resource
		errs.Go(func() error {
			err := parallelResourceDeletion(resourceMap, resource, config.Timeout)
			if err != nil {
				return err
			}
			return nil
		})
	}

	// Wait for all deletions to complete, and check for errors
	if err := errs.Wait(); err != nil {
		log.Fatal(err)
	}

	log.Printf("-- Deletion complete for project %v --\n", config.Project)
}

func parallelResourceDeletion(resourceMap map[string]gcp.Resource, resource gcp.Resource, dependencyTimeout int) error {

	// deleteSuccess = false
	pollTime := 5
	seconds := 0

	// Wait for dependencies to delete
	for _, dependencyResourceName := range resource.Dependencies() {
		if seconds > dependencyTimeout {
			return fmt.Errorf("[Error] Resource %v timed out whilst waiting for dependency %v to delete. Time waited: %v", resource, dependencyResourceName, dependencyTimeout)
		}
		dependencyResource := resourceMap[dependencyResourceName]
		if len(dependencyResource.List()) != 0 {
			time.Sleep(time.Duration(pollTime) * time.Second)
			seconds += pollTime
			log.Printf("[Waiting] Resource %v waiting for dependency %v to delete. Time waited: %v\n", resource.Name(), dependencyResource.Name(), seconds)
		} else {
			break
		}
	}

	return resource.Remove()
}
