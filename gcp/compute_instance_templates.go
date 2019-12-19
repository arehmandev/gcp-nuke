package gcp

import (
	"fmt"
	"log"
	"time"

	"github.com/arehmandev/gcp-nuke/config"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/compute/v1"
)

// ComputeInstanceTemplates -
type ComputeInstanceTemplates struct {
	serviceClient *compute.Service
	base          ResourceBase
	resourceMap   map[string]DefaultResourceProperties
}

func init() {
	client := clientFactory(compute.ComputeScope)

	computeService, err := compute.New(client)
	if err != nil {
		log.Fatal(err)
	}
	computeResource := ComputeInstanceTemplates{
		serviceClient: computeService,
	}
	register(&computeResource)
}

// Name - Name of the resourceLister for ComputeInstanceTemplates
func (c *ComputeInstanceTemplates) Name() string {
	return "ComputeInstanceTemplates"
}

// ToSlice - Name of the resourceLister for ComputeInstanceTemplates
func (c *ComputeInstanceTemplates) ToSlice() (slice []string) {
	for key := range c.resourceMap {
		slice = append(slice, key)
	}
	return slice
}

// Setup - populates the struct
func (c *ComputeInstanceTemplates) Setup(config config.Config) {
	c.base.config = config
	c.resourceMap = make(map[string]DefaultResourceProperties)

}

// List - Returns a list of all ComputeInstanceTemplates
func (c *ComputeInstanceTemplates) List(refreshCache bool) []string {
	if !refreshCache {
		return c.ToSlice()
	}
	log.Println("[Info] Retrieving list of resources for", c.Name())
	instanceListCall := c.serviceClient.InstanceTemplates.List(c.base.config.Project)
	instanceList, err := instanceListCall.Do()
	if err != nil {
		log.Fatal(err)
	}

	for _, instance := range instanceList.Items {
		instanceResource := DefaultResourceProperties{}
		c.resourceMap[instance.Name] = instanceResource
	}
	return c.ToSlice()
}

// Dependencies - Returns a List of resource names to check for
func (c *ComputeInstanceTemplates) Dependencies() []string {
	a := ComputeInstanceRegionGroups{}
	b := ComputeInstanceZoneGroups{}
	return []string{a.Name(), b.Name()}
}

// Remove -
func (c *ComputeInstanceTemplates) Remove() error {

	// Removal logic
	errs, _ := errgroup.WithContext(c.base.config.Context)

	for instanceID := range c.resourceMap {
		instanceID := instanceID

		// Parallel instance deletion
		errs.Go(func() error {
			deleteCall := c.serviceClient.InstanceTemplates.Delete(c.base.config.Project, instanceID)
			operation, err := deleteCall.Do()
			if err != nil {
				return err
			}
			var opStatus string
			seconds := 0
			for opStatus != "DONE" {
				log.Printf("[Info] Resource currently being deleted %v [type: %v project: %v] (%v seconds)", instanceID, c.Name(), c.base.config.Project, seconds)
				operationCall := c.serviceClient.GlobalOperations.Get(c.base.config.Project, operation.Name)
				checkOpp, err := operationCall.Do()
				if err != nil {
					return err
				}
				opStatus = checkOpp.Status

				time.Sleep(time.Duration(c.base.config.PollTime) * time.Second)
				seconds += c.base.config.PollTime
				if seconds > c.base.config.Timeout {
					return fmt.Errorf("[Error] Resource deletion timed out for %v [type: %v project: %v] (%v seconds)", instanceID, c.Name(), c.base.config.Project, c.base.config.Timeout)
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
