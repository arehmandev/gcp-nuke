package gcp

import (
	"log"

	"github.com/arehmandev/gcp-nuke/config"
	"github.com/arehmandev/gcp-nuke/helpers"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
)

// ComputeDisks -
type ComputeDisks struct {
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
	computeResource := ComputeDisks{
		serviceClient: computeService,
	}
	register(&computeResource)
}

// Name - Name of the resourceLister for ComputeDisks
func (c *ComputeDisks) Name() string {
	return "ComputeDisks"
}

// Setup - populates the struct
func (c *ComputeDisks) Setup(config config.Config) {
	c.base.config = config
	c.base.resourceMap = make(map[string]string)
	c.List(true)
}

// List - Returns a list of all ComputeDisks
func (c *ComputeDisks) List(refreshCache bool) []string {
	if !refreshCache {
		return helpers.MapKeys(c.base.resourceMap)
	}
	log.Println("[Info] Retrieving list of resources for", c.Name())
	for _, zone := range c.base.config.Zones {
		instanceListCall := c.serviceClient.Disks.List(c.base.config.Project, zone)
		instanceList, err := instanceListCall.Do()
		if err != nil {
			log.Fatal(err)
		}

		for _, instance := range instanceList.Items {
			c.base.resourceMap[instance.Name] = zone
		}
	}
	return helpers.MapKeys(c.base.resourceMap)
}

// Dependencies - Returns a List of resource names to check for
func (c *ComputeDisks) Dependencies() []string {
	a := ComputeInstances{}
	return []string{
		a.Name(),
	}
}

// Remove -
func (c *ComputeDisks) Remove() error {
	// Removal logic
	c.base.resourceMap = make(map[string]string)
	return nil
}
