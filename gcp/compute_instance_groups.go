package gcp

import (
	"log"

	"github.com/arehmandev/gcp-nuke/config"
	"github.com/arehmandev/gcp-nuke/helpers"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
)

// ComputeInstanceGroups -
type ComputeInstanceGroups struct {
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
	computeResource := ComputeInstanceGroups{
		serviceClient: computeService,
	}
	register(&computeResource)
}

// Name - Name of the resourceLister for ComputeInstanceGroups
func (c *ComputeInstanceGroups) Name() string {
	return "ComputeInstanceGroups"
}

// Setup - populates the struct
func (c *ComputeInstanceGroups) Setup(config config.Config) {
	log.Println("[Setup] Getting list for", c.Name())
	c.base.config = config
	c.List()
	c.base.cache = true
}

// List - Returns a list of all ComputeInstanceGroups
func (c *ComputeInstanceGroups) List() []string {
	if c.base.cache {
		return c.base.resourceNames
	}
	zoneListCall := c.serviceClient.Zones.List(c.base.config.Project)
	zoneList, err := zoneListCall.Do()
	if err != nil {
		log.Fatal(err)
	}

	for _, zone := range zoneList.Items {
		instanceListCall := c.serviceClient.InstanceGroups.List(c.base.config.Project, zone.Name)
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
func (c *ComputeInstanceGroups) Dependencies() []string {
	return nil
}

// Remove -
func (c *ComputeInstanceGroups) Remove() error {
	if len(c.base.resourceNames) == 0 {
		log.Println("[Skipping] No", c.Name(), "items to delete")
		return nil
	}
	log.Println("[Remove] Removing", c.Name(), "items:", c.List())
	// Removal logic
	c.base.resourceNames = []string{}
	return nil
}
