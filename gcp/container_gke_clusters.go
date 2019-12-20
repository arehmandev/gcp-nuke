package gcp

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/arehmandev/gcp-nuke/config"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/container/v1"
)

// ContainerGKEClusters -
type ContainerGKEClusters struct {
	serviceClient *container.Service
	base          ResourceBase
	resourceMap   map[string]DefaultResourceProperties
}

func init() {
	containerService, err := container.NewService(Ctx)
	if err != nil {
		log.Fatal(err)
	}
	containerResource := ContainerGKEClusters{
		serviceClient: containerService,
	}
	register(&containerResource)
}

// Name - Name of the resourceLister for ContainerGKEClusters
func (c *ContainerGKEClusters) Name() string {
	return "ContainerGKEClusters"
}

// ToSlice - Name of the resourceLister for ContainerGKEClusters
func (c *ContainerGKEClusters) ToSlice() (slice []string) {
	for key := range c.resourceMap {
		slice = append(slice, key)
	}
	return slice
}

// Setup - populates the struct
func (c *ContainerGKEClusters) Setup(config config.Config) {
	c.base.config = config
	c.resourceMap = make(map[string]DefaultResourceProperties)

}

// List - Returns a list of all ContainerGKEClusters
func (c *ContainerGKEClusters) List(refreshCache bool) []string {
	if !refreshCache {
		return c.ToSlice()
	}
	log.Println("[Info] Retrieving list of resources for", c.Name())
	instanceListCall := c.serviceClient.Projects.Locations.Clusters.List(fmt.Sprintf("projects/%v/locations/-", c.base.config.Project))
	instanceList, err := instanceListCall.Do()
	if err != nil {
		log.Fatal(err)
	}

	for _, instance := range instanceList.Clusters {
		instanceResource := DefaultResourceProperties{}
		clusterLink := extractGKESelfLink(instance.SelfLink)
		c.resourceMap[clusterLink] = instanceResource
	}

	return c.ToSlice()
}

// Dependencies - Returns a List of resource names to check for
func (c *ContainerGKEClusters) Dependencies() []string {
	return []string{}
}

// Remove -
func (c *ContainerGKEClusters) Remove() error {

	// Removal logic
	errs, _ := errgroup.WithContext(c.base.config.Context)

	for instanceID := range c.resourceMap {
		instanceID := instanceID
		location := strings.Split(instanceID, "/")[3]
		// Parallel instance deletion
		errs.Go(func() error {
			deleteCall := c.serviceClient.Projects.Locations.Clusters.Delete(instanceID)
			operation, err := deleteCall.Do()
			if err != nil {
				return err
			}
			var opStatus string
			seconds := 0
			for opStatus != "DONE" {
				log.Printf("[Info] Resource currently being deleted %v [type: %v project: %v] (%v seconds)", instanceID, c.Name(), c.base.config.Project, seconds)
				operationCall := c.serviceClient.Projects.Locations.Operations.Get(fmt.Sprintf("projects/%v/locations/%v/operations/%v", c.base.config.Project, location, operation.Name))
				checkOpp, err := operationCall.Do()
				if err != nil {
					return err
				}
				opStatus = checkOpp.Status

				time.Sleep(time.Duration(c.base.config.PollTime) * time.Second)
				seconds += c.base.config.PollTime
				if seconds > c.base.config.Timeout {
					return fmt.Errorf("[Error] Resource deletion timed out for %v [type: %v project: %v] (%v seconds):\n %v", instanceID, c.Name(), c.base.config.Project, c.base.config.Timeout, err.Error())
				}
			}
			delete(c.resourceMap, instanceID)
			log.Printf("[Info] Resource deleted %v [type: %v project: %v] (%v seconds)", instanceID, c.Name(), c.base.config.Project, seconds)
			return nil
		})

	}
	// Wait for all deletions to complete, and return the first non nil error
	err := errs.Wait()
	return err
}
