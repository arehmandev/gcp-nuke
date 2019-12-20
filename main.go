package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/arehmandev/gcp-nuke/config"
	"github.com/arehmandev/gcp-nuke/gcp"
	"golang.org/x/sync/errgroup"
)

func main() {

	// Behaviour to delete one project at a time - will be made into loop later
	project := os.Getenv("GCP_PROJECT_ID")
	if project == "" {
		log.Fatalln("GCP_PROJECT_ID environment variable not set")
	}
	config := config.Config{
		Project:  project,
		DryRun:   false,
		Timeout:  300,
		PollTime: 10,
		Context:  gcp.Ctx,
		Zones:    gcp.GetZones(gcp.Ctx, project),
		Regions:  gcp.GetRegions(gcp.Ctx, project),
	}
	log.Printf("[Info] Timeout %v seconds. Polltime %v seconds", config.Timeout, config.PollTime)
	removeProject(config)
}

func removeProject(config config.Config) {
	resourceMap := gcp.GetResourceMap(config)

	// Parallel deletion
	errs, _ := errgroup.WithContext(config.Context)

	for _, resource := range resourceMap {
		resource := resource
		errs.Go(func() error {
			resource.List(true)
			if config.DryRun {
				parallelDryRun(resourceMap, resource, config)
				return nil
			}
			err := parallelResourceDeletion(resourceMap, resource, config)

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

	log.Printf("-- Deletion complete for project %v (dry-run: %v) --\n", config.Project, config.DryRun)
}

func parallelResourceDeletion(resourceMap map[string]gcp.Resource, resource gcp.Resource, config config.Config) error {
	refreshCache := false
	if len(resource.List(false)) == 0 {
		log.Println("[Skipping] No", resource.Name(), "items to delete")
		return nil
	}

	timeOut := config.Timeout
	pollTime := config.PollTime
	seconds := 0

	// Wait for dependencies to delete
	for _, dependencyResourceName := range resource.Dependencies() {
		if seconds > timeOut {
			return fmt.Errorf("[Error] Resource %v timed out whilst waiting for dependency %v to delete. (%v seconds)", resource.Name(), dependencyResourceName, timeOut)
		}
		dependencyResource := resourceMap[dependencyResourceName]
		if len(dependencyResource.List(false)) != 0 {
			refreshCache = true
			time.Sleep(time.Duration(pollTime) * time.Second)
			seconds += pollTime
			log.Printf("[Waiting] Resource %v waiting for dependency %v to delete. (%v seconds)\n", resource.Name(), dependencyResource.Name(), seconds)
		} else {
			break
		}
	}

	if refreshCache {
		resource.List(refreshCache)
	}

	log.Println("[Remove] Removing", resource.Name(), "items:", resource.List(false))
	seconds = 0
	err := resource.Remove()

	// Unfortunately the API seems inconsistent with timings, so retry until any dependent resources delete
	for apiErrorCheck(err) {

		if seconds > timeOut {
			return fmt.Errorf("[Error] Resource %v timed out whilst trying to delete. (%v seconds). Details of error below:\n %v", resource.Name(), timeOut, err.Error())
		}

		log.Printf("[Remove] In use Resource: %v. Items: %v. Waiting before retrying delete. (%v seconds)", resource.Name(), resource.List(false), seconds)
		time.Sleep(time.Duration(pollTime) * time.Second)
		seconds += pollTime
		err = resource.Remove()

	}

	// Add some info to it
	if err != nil {
		detailedError := fmt.Errorf("[Error] Resource: %v. Items: %v. Details of error below:\n %v", resource.Name(), resource.List(false), err.Error())
		err = detailedError
	}

	return err
}

func apiErrorCheck(err error) bool {
	if err == nil {
		return false
	}
	errorDescriptors := []string{"resourceInUseByAnotherResource", "resourceNotReady"}
	for _, errorDesc := range errorDescriptors {
		if strings.Contains(err.Error(), errorDesc) {
			return true
		}
	}
	return false
}
