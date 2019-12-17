package gcp

import (
	"log"

	"github.com/arehmandev/gcp-nuke/config"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
)

// ComputeInstances -
type ComputeInstances struct {
	resourceNames []string
	serviceClient *compute.Service
	config        config.Config
	removed       bool
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
	log.Println("[Setup] Getting list for", c.Name())
	c.config = config
	c.List()
	if len(c.resourceNames) == 0 {
		c.removed = true
	}
}

// List - Returns a list of all ComputeInstances
func (c *ComputeInstances) List() []string {
	for _, project := range c.config.Projects {
		zoneListCall := c.serviceClient.Zones.List(project)
		zoneList, err := zoneListCall.Do()
		if err != nil {
			log.Fatal(err)
		}

		for _, zone := range zoneList.Items {

			instanceListCall := c.serviceClient.Instances.List(project, zone.Name)
			instanceList, err := instanceListCall.Do()
			if err != nil {
				log.Fatal(err)
			}

			for _, instance := range instanceList.Items {
				c.resourceNames = append(c.resourceNames, instance.Name)
			}
		}
	}
	return c.resourceNames
}

// Dependencies - Returns a List of resource names to check for
func (c *ComputeInstances) Dependencies() []string {
	return nil
}

// Remove -
func (c *ComputeInstances) Remove() error {
	if c.removed {
		return nil
	}
	log.Println("[Remove] Removing", c.Name(), "items:", c.List())
	// Removal logic
	c.removed = true
	return nil
}
