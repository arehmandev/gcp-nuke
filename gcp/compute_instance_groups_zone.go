package gcp

import (
	"fmt"
	"log"
	"reflect"
	"sync"
	"time"

	"github.com/BESTSELLER/gcp-nuke/config"
	"github.com/BESTSELLER/gcp-nuke/helpers"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/syncmap"
	"google.golang.org/api/compute/v1"
)

// ComputeInstanceGroupsZone -
type ComputeInstanceGroupsZone struct {
	serviceClient *compute.Service
	// Required to skip gke nodepools
	gkeInstanceGroups []string
	base              ResourceBase
	resourceMap       syncmap.Map
}

func init() {
	computeService, err := compute.NewService(Ctx)
	if err != nil {
		log.Fatal(err)
	}
	computeResource := ComputeInstanceGroupsZone{
		serviceClient: computeService,
	}
	register(&computeResource)
}

// Name - Name of the resourceLister for ComputeInstanceGroupsZone
func (c *ComputeInstanceGroupsZone) Name() string {
	return "ComputeInstanceGroupsZone"
}

// ToSlice - Name of the resourceLister for ComputeInstanceGroupsZone
func (c *ComputeInstanceGroupsZone) ToSlice() (slice []string) {
	return helpers.SortedSyncMapKeys(&c.resourceMap)

}

// Setup - populates the struct
func (c *ComputeInstanceGroupsZone) Setup(config config.Config) {
	c.base.config = config

	// Get the node pool list with some reflection rather than re-instantiating
	a := ContainerGKEClusters{}
	gkeResource := resourceMap[a.Name()]
	gkeInstance := reflect.ValueOf(gkeResource).Elem().Addr().Interface().(*ContainerGKEClusters)
	gkeInstance.Setup(config)
	gkeInstance.List(true)
	c.gkeInstanceGroups = gkeInstance.InstanceGroups
}

// List - Returns a list of all ComputeInstanceGroupsZone
func (c *ComputeInstanceGroupsZone) List(refreshCache bool) []string {
	if !refreshCache {
		return c.ToSlice()
	}
	// Refresh resource map
	c.resourceMap = sync.Map{}

	for _, zone := range c.base.config.Zones {
		instanceListCall := c.serviceClient.InstanceGroupManagers.List(c.base.config.Project, zone)
		instanceList, err := instanceListCall.Do()
		if err != nil {
			log.Fatal(err)
		}

		for _, instance := range instanceList.Items {

			if helpers.SliceContains(c.gkeInstanceGroups, instance.Name) {
				continue
			}

			instanceResource := DefaultResourceProperties{
				zone: zone,
			}
			c.resourceMap.Store(instance.Name, instanceResource)
		}
	}
	return c.ToSlice()
}

// Dependencies - Returns a List of resource names to check for
func (c *ComputeInstanceGroupsZone) Dependencies() []string {
	a := ComputeZoneAutoScalers{}
	return []string{a.Name()}
}

// Remove -
func (c *ComputeInstanceGroupsZone) Remove() error {

	// Removal logic
	errs, _ := errgroup.WithContext(c.base.config.Context)

	c.resourceMap.Range(func(key, value interface{}) bool {
		instanceID := key.(string)
		zone := value.(DefaultResourceProperties).zone

		// Parallel instance deletion
		errs.Go(func() error {
			deleteCall := c.serviceClient.InstanceGroupManagers.Delete(c.base.config.Project, zone, instanceID)
			operation, err := deleteCall.Do()
			if err != nil {
				return err
			}
			var opStatus string
			seconds := 0
			for opStatus != "DONE" {
				log.Printf("[Info] Resource currently being deleted %v [type: %v project: %v zone: %v] (%v seconds)", instanceID, c.Name(), c.base.config.Project, zone, seconds)

				operationCall := c.serviceClient.ZoneOperations.Get(c.base.config.Project, zone, operation.Name)
				checkOpp, err := operationCall.Do()
				if err != nil {
					return err
				}
				opStatus = checkOpp.Status

				time.Sleep(time.Duration(c.base.config.PollTime) * time.Second)
				seconds += c.base.config.PollTime
				if seconds > c.base.config.Timeout {
					return fmt.Errorf("[Error] Resource deletion timed out for %v [type: %v project: %v zone: %v] (%v seconds)", instanceID, c.Name(), c.base.config.Project, zone, c.base.config.Timeout)
				}
			}
			c.resourceMap.Delete(instanceID)

			log.Printf("[Info] Resource deleted %v [type: %v project: %v zone: %v] (%v seconds)", instanceID, c.Name(), c.base.config.Project, zone, seconds)
			return nil
		})
		return true
	})
	// Wait for all deletions to complete, and return the first non nil error
	err := errs.Wait()
	return err
}
