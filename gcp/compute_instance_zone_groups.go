package gcp

import (
	"fmt"
	"log"
	"time"

	"github.com/arehmandev/gcp-nuke/config"
	"golang.org/x/oauth2/google"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/compute/v1"
)

// ComputeInstanceZoneGroups -
type ComputeInstanceZoneGroups struct {
	serviceClient *compute.Service
	base          ResourceBase
	resourceMap   map[string]DefaultResourceProperties
}

func init() {
	client, err := google.DefaultClient(ctx, compute.ComputeScope)
	if err != nil {
		log.Fatal(err)
	}
	computeService, err := compute.New(client)
	if err != nil {
		log.Fatal(err)
	}
	computeResource := ComputeInstanceZoneGroups{
		serviceClient: computeService,
	}
	register(&computeResource)
}

// Name - Name of the resourceLister for ComputeInstanceZoneGroups
func (c *ComputeInstanceZoneGroups) Name() string {
	return "ComputeInstanceZoneGroups"
}

// ToSlice - Name of the resourceLister for ComputeInstanceZoneGroups
func (c *ComputeInstanceZoneGroups) ToSlice() (slice []string) {
	for key := range c.resourceMap {
		slice = append(slice, key)
	}
	return slice
}

// Setup - populates the struct
func (c *ComputeInstanceZoneGroups) Setup(config config.Config) {
	c.base.config = config
	c.resourceMap = make(map[string]DefaultResourceProperties)
	c.List(true)
}

// List - Returns a list of all ComputeInstanceZoneGroups
func (c *ComputeInstanceZoneGroups) List(refreshCache bool) []string {
	if !refreshCache {
		return c.ToSlice()
	}
	log.Println("[Info] Retrieving list of resources for", c.Name())
	for _, zone := range c.base.config.Zones {
		instanceListCall := c.serviceClient.InstanceGroupManagers.List(c.base.config.Project, zone)
		instanceList, err := instanceListCall.Do()
		if err != nil {
			log.Fatal(err)
		}

		for _, instance := range instanceList.Items {
			instanceResource := DefaultResourceProperties{
				zone: zone,
			}
			c.resourceMap[instance.Name] = instanceResource
		}
	}
	return c.ToSlice()
}

// Dependencies - Returns a List of resource names to check for
func (c *ComputeInstanceZoneGroups) Dependencies() []string {
	return []string{}
}

// Remove -
func (c *ComputeInstanceZoneGroups) Remove() error {

	// Removal logic
	errs, _ := errgroup.WithContext(c.base.config.Context)

	for instanceID, instanceProperties := range c.resourceMap {
		instanceID := instanceID
		zone := instanceProperties.zone

		// Parallel instance deletion
		errs.Go(func() error {
			deleteCall := c.serviceClient.InstanceGroupManagers.Delete(c.base.config.Project, zone, instanceID)
			var opStatus string
			seconds := 0
			for opStatus != "DONE" {
				log.Printf("[Info] Resource currently being deleted %v [type: %v project: %v zone: %v] (%v seconds)", instanceID, c.Name(), c.base.config.Project, zone, seconds)
				operation, err := deleteCall.Do()
				if err != nil {
					return err
				}
				opStatus = operation.Status

				time.Sleep(time.Duration(c.base.config.PollTime) * time.Second)
				seconds += c.base.config.PollTime
				if seconds > c.base.config.Timeout {
					return fmt.Errorf("[Error] Resource deletion timed out for %v [type: %v project: %v zone: %v] (%v seconds)", instanceID, c.Name(), c.base.config.Project, zone, c.base.config.Timeout)
				}
			}
			delete(c.resourceMap, instanceID)
			log.Printf("[Info] Resource deleted %v [type: %v project: %v zone: %v] (%v seconds)", instanceID, c.Name(), c.base.config.Project, zone, seconds)
			return nil
		})

	}
	// Wait for all deletions to complete, and return the first non nil error
	err := errs.Wait()
	return err
}
