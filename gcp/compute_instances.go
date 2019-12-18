package gcp

import (
	"log"

	"github.com/arehmandev/gcp-nuke/config"
	"github.com/arehmandev/gcp-nuke/helpers"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
)

// ComputeInstances -
type ComputeInstances struct {
	serviceClient *compute.Service
	base          ResourceBase
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
	computeResource := ComputeInstances{
		serviceClient: computeService,
	}
	register(&computeResource)
}

// Name - Name of the resourceLister for ComputeInstances
func (c *ComputeInstances) Name() string {
	return "ComputeInstances"
}

// Setup - populates the struct
func (c *ComputeInstances) Setup(config config.Config) {
	c.base.config = config
	c.List(true)
}

// List - Returns a list of all ComputeInstances
func (c *ComputeInstances) List(refreshCache bool) []string {
	if !refreshCache {
		return c.base.resourceNames
	}
	log.Println("Retrieving list of resources for", c.Name())
	for _, zone := range c.base.config.Zones {
		instanceListCall := c.serviceClient.Instances.List(c.base.config.Project, zone)
		instanceList, err := instanceListCall.Do()
		if err != nil {
			log.Fatal(err)
		}

		for _, instance := range instanceList.Items {
			if !helpers.SliceContains(c.base.resourceNames, instance.Name) {
				c.base.resourceNames = append(c.base.resourceNames, instance.Name)
			}
		}
	}
	return c.base.resourceNames
}

// Dependencies - Returns a List of resource names to check for
func (c *ComputeInstances) Dependencies() []string {
	a := ComputeInstanceGroups{}
	return []string{
		a.Name(),
	}
}

// Remove -
func (c *ComputeInstances) Remove() error {
	// Removal logic
	// for _, zone := range c.base.config.Zones {
	// 	for _, instanceid := range c.base.resourceNames {
	// 		deleteCall := c.serviceClient.Instances.Delete(c.base.config.Project, zone, instanceid)
	// 		deleteCall.Do()
	// 	}
	// }

	c.base.resourceNames = []string{}
	return nil
}
