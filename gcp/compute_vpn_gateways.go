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

// ComputeVPNGateways -
type ComputeVPNGateways struct {
	serviceClient *compute.Service
	base          ResourceBase
	resourceMap   syncmap.Map
}

func init() {
	computeService, err := compute.NewService(Ctx)
	if err != nil {
		log.Fatal(err)
	}
	computeResource := ComputeVPNGateways{
		serviceClient: computeService,
	}
	register(&computeResource)
}

// Name - Name of the resourceLister for ComputeVPNGateways
func (c *ComputeVPNGateways) Name() string {
	return "ComputeVPNGateways"
}

// ToSlice - Name of the resourceLister for ComputeVPNGateways
func (c *ComputeVPNGateways) ToSlice() (slice []string) {
	return helpers.SortedSyncMapKeys(&c.resourceMap)

}

// Setup - populates the struct
func (c *ComputeVPNGateways) Setup(config config.Config) {
	c.base.config = config

}

// List - Returns a list of all ComputeVPNGateways
func (c *ComputeVPNGateways) List(refreshCache bool) []string {
	if !refreshCache {
		return c.ToSlice()
	}
	// Refresh resource map
	c.resourceMap = sync.Map{}

	for _, region := range c.base.config.Regions {
		gatewayListCall := c.serviceClient.VpnGateways.List(c.base.config.Project, region)
		gatewayList, err := gatewayListCall.Do()
		if err != nil {
			log.Fatal(err)
		}

		for _, gateway := range gatewayList.Items {
			c.resourceMap.Store(gateway.Name, region)
		}
	}
	return c.ToSlice()
}

// Dependencies - Returns a List of resource names to check for
func (c *ComputeVPNGateways) Dependencies() []string {
	a := ComputeVPNTunnels{}
	return []string{a.Name()}
}

// Remove -
func (c *ComputeVPNGateways) Remove() error {

	// Removal logic
	errs, _ := errgroup.WithContext(c.base.config.Context)

	c.resourceMap.Range(func(key, value interface{}) bool {
		gatewayID := key.(string)
		region := value.(string)

		// Parallel gateway deletion
		errs.Go(func() error {
			deleteCall := c.serviceClient.VpnGateways.Delete(c.base.config.Project, region, gatewayID)
			operation, err := deleteCall.Do()
			if err != nil {
				return err
			}
			var opStatus string
			seconds := 0
			for opStatus != "DONE" {
				log.Printf("[Info] Resource currently being deleted %v [type: %v project: %v] (%v seconds)", gatewayID, c.Name(), c.base.config.Project, seconds)

				operationCall := c.serviceClient.RegionOperations.Get(c.base.config.Project, region, operation.Name)
				checkOpp, err := operationCall.Do()
				if err != nil {
					return err
				}
				opStatus = checkOpp.Status

				time.Sleep(time.Duration(c.base.config.PollTime) * time.Second)
				seconds += c.base.config.PollTime
				if seconds > c.base.config.Timeout {
					return fmt.Errorf("[Error] Resource deletion timed out for %v [type: %v project: %v] (%v seconds)", gatewayID, c.Name(), c.base.config.Project, c.base.config.Timeout)
				}
			}
			c.resourceMap.Delete(gatewayID)

			log.Printf("[Info] Resource deleted %v [type: %v project: %v] (%v seconds)", gatewayID, c.Name(), c.base.config.Project, seconds)
			return nil
		})
		return true
	})
	// Wait for all deletions to complete, and return the first non nil error
	err := errs.Wait()
	return err
}
