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
	resourceNames []string
	serviceClient *compute.Service
	config        config.Config
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
	log.Println("[Setup] Getting list for", c.Name())
	c.config = config
	c.List()
}

// List - Returns a list of all ComputeDisks
func (c *ComputeDisks) List() []string {
	zoneListCall := c.serviceClient.Zones.List(c.config.Project)
	zoneList, err := zoneListCall.Do()
	if err != nil {
		log.Fatal(err)
	}

	for _, zone := range zoneList.Items {
		instanceListCall := c.serviceClient.Disks.List(c.config.Project, zone.Name)
		instanceList, err := instanceListCall.Do()
		if err != nil {
			log.Fatal(err)
		}

		for _, instance := range instanceList.Items {
			if !helpers.SliceContains(c.resourceNames, instance.Name) {
				c.resourceNames = append(c.resourceNames, instance.Name)
			}
		}
	}
	return c.resourceNames
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
	if len(c.resourceNames) == 0 {
		return nil
	}
	log.Println("[Remove] Removing", c.Name(), "items:", c.List())
	// Removal logic
	c.resourceNames = []string{}
	return nil
}
