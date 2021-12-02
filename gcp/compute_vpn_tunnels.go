package gcp

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/BESTSELLER/gcp-nuke/config"
	"github.com/BESTSELLER/gcp-nuke/helpers"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/syncmap"
	"google.golang.org/api/compute/v1"
)

// ComputeVPNTunnels -
type ComputeVPNTunnels struct {
	serviceClient *compute.Service
	base          ResourceBase
	resourceMap   syncmap.Map
}

func init() {
	computeService, err := compute.NewService(Ctx)
	if err != nil {
		log.Fatal(err)
	}
	computeResource := ComputeVPNTunnels{
		serviceClient: computeService,
	}
	register(&computeResource)
}

// Name - Name of the resourceLister for ComputeVPNTunnels
func (c *ComputeVPNTunnels) Name() string {
	return "ComputeVPNTunnels"
}

// ToSlice - Name of the resourceLister for ComputeVPNTunnels
func (c *ComputeVPNTunnels) ToSlice() (slice []string) {
	return helpers.SortedSyncMapKeys(&c.resourceMap)

}

// Setup - populates the struct
func (c *ComputeVPNTunnels) Setup(config config.Config) {
	c.base.config = config

}

// List - Returns a list of all ComputeVPNTunnels
func (c *ComputeVPNTunnels) List(refreshCache bool) []string {
	if !refreshCache {
		return c.ToSlice()
	}
	// Refresh resource map
	c.resourceMap = sync.Map{}

	for _, region := range c.base.config.Regions {
		tunnelListCall := c.serviceClient.VpnTunnels.List(c.base.config.Project, region)
		tunnelList, err := tunnelListCall.Do()
		if err != nil {
			log.Fatal(err)
		}

		for _, tunnel := range tunnelList.Items {
			c.resourceMap.Store(tunnel.Name, region)
		}
	}
	return c.ToSlice()
}

// Dependencies - Returns a List of resource names to check for
func (c *ComputeVPNTunnels) Dependencies() []string {
	return []string{}
}

// Remove -
func (c *ComputeVPNTunnels) Remove() error {

	// Removal logic
	errs, _ := errgroup.WithContext(c.base.config.Context)

	c.resourceMap.Range(func(key, value interface{}) bool {
		tunnelID := key.(string)
		region := value.(string)

		// Parallel tunnel deletion
		errs.Go(func() error {
			deleteCall := c.serviceClient.VpnTunnels.Delete(c.base.config.Project, region, tunnelID)
			operation, err := deleteCall.Do()
			if err != nil {
				return err
			}
			var opStatus string
			seconds := 0
			for opStatus != "DONE" {
				log.Printf("[Info] Resource currently being deleted %v [type: %v project: %v] (%v seconds)", tunnelID, c.Name(), c.base.config.Project, seconds)

				operationCall := c.serviceClient.RegionOperations.Get(c.base.config.Project, region, operation.Name)
				checkOpp, err := operationCall.Do()
				if err != nil {
					return err
				}
				opStatus = checkOpp.Status

				time.Sleep(time.Duration(c.base.config.PollTime) * time.Second)
				seconds += c.base.config.PollTime
				if seconds > c.base.config.Timeout {
					return fmt.Errorf("[Error] Resource deletion timed out for %v [type: %v project: %v] (%v seconds)", tunnelID, c.Name(), c.base.config.Project, c.base.config.Timeout)
				}
			}
			c.resourceMap.Delete(tunnelID)

			log.Printf("[Info] Resource deleted %v [type: %v project: %v] (%v seconds)", tunnelID, c.Name(), c.base.config.Project, seconds)
			return nil
		})
		return true
	})
	// Wait for all deletions to complete, and return the first non nil error
	err := errs.Wait()
	return err
}
