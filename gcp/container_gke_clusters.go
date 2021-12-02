package gcp

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/BESTSELLER/gcp-nuke/config"
	"github.com/BESTSELLER/gcp-nuke/helpers"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/syncmap"
	"google.golang.org/api/container/v1"
)

// ContainerGKEClusters -
type ContainerGKEClusters struct {
	serviceClient  *container.Service
	base           ResourceBase
	resourceMap    syncmap.Map
	InstanceGroups []string
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
	return helpers.SortedSyncMapKeys(&c.resourceMap)

}

// Setup - populates the struct
func (c *ContainerGKEClusters) Setup(config config.Config) {
	c.base.config = config

}

// List - Returns a list of all ContainerGKEClusters
func (c *ContainerGKEClusters) List(refreshCache bool) []string {
	if !refreshCache {
		return c.ToSlice()
	}
	// Refresh resource map
	c.resourceMap = sync.Map{}

	instanceListCall := c.serviceClient.Projects.Locations.Clusters.List(fmt.Sprintf("projects/%v/locations/-", c.base.config.Project))
	instanceList, err := instanceListCall.Do()
	if err != nil {
		log.Fatal(err)
	}

	for _, instance := range instanceList.Clusters {
		c.appendInstanceGroups(instance.Name, instance.Location)
		instanceResource := DefaultResourceProperties{}
		clusterLink := extractGKESelfLink(instance.SelfLink)
		c.resourceMap.Store(clusterLink, instanceResource)
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

	c.resourceMap.Range(func(key, value interface{}) bool {
		instanceID := key.(string)
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
			c.resourceMap.Delete(instanceID)

			log.Printf("[Info] Resource deleted %v [type: %v project: %v] (%v seconds)", instanceID, c.Name(), c.base.config.Project, seconds)
			return nil
		})
		return true
	})
	// Wait for all deletions to complete, and return the first non nil error
	err := errs.Wait()
	return err
}

// appendInstanceGroups - keep track of instance groups - this is used by compute_instance_zone_groups to exclude any gke nodepools
func (c *ContainerGKEClusters) appendInstanceGroups(clusterName, clusterLocation string) {
	parentLocation := fmt.Sprintf("projects/%v/locations/%v/clusters/%v", c.base.config.Project, clusterLocation, clusterName)
	nodePoolCall := c.serviceClient.Projects.Locations.Clusters.NodePools.List(parentLocation)
	nodePools, err := nodePoolCall.Do()
	if err != nil {
		log.Fatal((err))
	}
	for _, nodePool := range nodePools.NodePools {
		for _, instanceGroupURL := range nodePool.InstanceGroupUrls {
			instanceGroupName := strings.Split(instanceGroupURL, "/instanceGroupManagers/")[1]
			c.InstanceGroups = append(c.InstanceGroups, instanceGroupName)
		}
	}
}
