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

// ComputeInstanceRegionGroups -
type ComputeInstanceRegionGroups struct {
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
	computeResource := ComputeInstanceRegionGroups{
		serviceClient: computeService,
	}
	register(&computeResource)
}

// Name - Name of the resourceLister for ComputeInstanceRegionGroups
func (c *ComputeInstanceRegionGroups) Name() string {
	return "ComputeInstanceRegionGroups"
}

// ToSlice - Name of the resourceLister for ComputeInstanceRegionGroups
func (c *ComputeInstanceRegionGroups) ToSlice() (slice []string) {
	for key := range c.resourceMap {
		slice = append(slice, key)
	}
	return slice
}

// Setup - populates the struct
func (c *ComputeInstanceRegionGroups) Setup(config config.Config) {
	c.base.config = config
	c.resourceMap = make(map[string]DefaultResourceProperties)
	c.List(true)
}

// List - Returns a list of all ComputeInstanceRegionGroups
func (c *ComputeInstanceRegionGroups) List(refreshCache bool) []string {
	if !refreshCache {
		return c.ToSlice()
	}
	log.Println("[Info] Retrieving list of resources for", c.Name())
	for _, region := range c.base.config.Regions {
		instanceListCall := c.serviceClient.RegionInstanceGroupManagers.List(c.base.config.Project, region)
		instanceList, err := instanceListCall.Do()
		if err != nil {
			log.Fatal(err)
		}

		for _, instance := range instanceList.Items {
			instanceResource := DefaultResourceProperties{
				region: region,
			}
			c.resourceMap[instance.Name] = instanceResource
		}
	}
	return c.ToSlice()
}

// Dependencies - Returns a List of resource names to check for
func (c *ComputeInstanceRegionGroups) Dependencies() []string {
	a := ComputeRegionAutoScalers{}
	return []string{a.Name()}
}

// Remove -
func (c *ComputeInstanceRegionGroups) Remove() error {

	// Removal logic
	errs, _ := errgroup.WithContext(c.base.config.Context)

	for instanceID, instanceProperties := range c.resourceMap {
		instanceID := instanceID
		region := instanceProperties.region

		// Parallel instance deletion
		errs.Go(func() error {
			deleteCall := c.serviceClient.RegionInstanceGroupManagers.Delete(c.base.config.Project, region, instanceID)
			var opStatus string
			seconds := 0
			for opStatus != "DONE" {
				log.Printf("[Info] Resource currently being deleted %v [type: %v project: %v region: %v] (%v seconds)", instanceID, c.Name(), c.base.config.Project, region, seconds)
				operation, err := deleteCall.Do()
				if err != nil {
					return err
				}
				opStatus = operation.Status

				time.Sleep(time.Duration(c.base.config.PollTime) * time.Second)
				seconds += c.base.config.PollTime
				if seconds > c.base.config.Timeout {
					return fmt.Errorf("[Error] Resource deletion timed out for %v [type: %v project: %v region: %v] (%v seconds)", instanceID, c.Name(), c.base.config.Project, region, c.base.config.Timeout)
				}
			}
			delete(c.resourceMap, instanceID)
			log.Printf("[Info] Resource deleted %v [type: %v project: %v region: %v] (%v seconds)", instanceID, c.Name(), c.base.config.Project, region, seconds)
			return nil
		})

	}
	// Wait for all deletions to complete, and return the first non nil error
	err := errs.Wait()
	return err
}
