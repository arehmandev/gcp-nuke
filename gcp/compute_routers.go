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

// ComputeRouters -
type ComputeRouters struct {
	serviceClient *compute.Service
	base          ResourceBase
	resourceMap   syncmap.Map
}

func init() {
	computeService, err := compute.NewService(Ctx)
	if err != nil {
		log.Fatal(err)
	}
	computeResource := ComputeRouters{
		serviceClient: computeService,
	}
	register(&computeResource)
}

// Name - Name of the resourceLister for ComputeRouters
func (c *ComputeRouters) Name() string {
	return "ComputeRouters"
}

// ToSlice - Name of the resourceLister for ComputeRouters
func (c *ComputeRouters) ToSlice() (slice []string) {
	return helpers.SortedSyncMapKeys(&c.resourceMap)

}

// Setup - populates the struct
func (c *ComputeRouters) Setup(config config.Config) {
	c.base.config = config

}

// List - Returns a list of all ComputeRouters
func (c *ComputeRouters) List(refreshCache bool) []string {
	if !refreshCache {
		return c.ToSlice()
	}
	// Refresh resource map
	c.resourceMap = sync.Map{}

	for _, region := range c.base.config.Regions {
		routerListCall := c.serviceClient.Routers.List(c.base.config.Project, region)
		routerList, err := routerListCall.Do()
		if err != nil {
			log.Fatal(err)
		}

		for _, router := range routerList.Items {
			c.resourceMap.Store(router.Name, region)
		}
	}
	return c.ToSlice()
}

// Dependencies - Returns a List of resource names to check for
func (c *ComputeRouters) Dependencies() []string {
	a := ComputeVPNTunnels{}
	b := ComputeVPNGateways{}
	return []string{a.Name(), b.Name()}
}

// Remove -
func (c *ComputeRouters) Remove() error {

	// Removal logic
	errs, _ := errgroup.WithContext(c.base.config.Context)

	c.resourceMap.Range(func(key, value interface{}) bool {
		routerID := key.(string)
		region := value.(string)

		// Parallel router deletion
		errs.Go(func() error {
			deleteCall := c.serviceClient.Routers.Delete(c.base.config.Project, region, routerID)
			operation, err := deleteCall.Do()
			if err != nil {
				return err
			}
			var opStatus string
			seconds := 0
			for opStatus != "DONE" {
				log.Printf("[Info] Resource currently being deleted %v [type: %v project: %v] (%v seconds)", routerID, c.Name(), c.base.config.Project, seconds)

				operationCall := c.serviceClient.RegionOperations.Get(c.base.config.Project, region, operation.Name)
				checkOpp, err := operationCall.Do()
				if err != nil {
					return err
				}
				opStatus = checkOpp.Status

				time.Sleep(time.Duration(c.base.config.PollTime) * time.Second)
				seconds += c.base.config.PollTime
				if seconds > c.base.config.Timeout {
					return fmt.Errorf("[Error] Resource deletion timed out for %v [type: %v project: %v] (%v seconds)", routerID, c.Name(), c.base.config.Project, c.base.config.Timeout)
				}
			}
			c.resourceMap.Delete(routerID)

			log.Printf("[Info] Resource deleted %v [type: %v project: %v] (%v seconds)", routerID, c.Name(), c.base.config.Project, seconds)
			return nil
		})
		return true
	})
	// Wait for all deletions to complete, and return the first non nil error
	err := errs.Wait()
	return err
}
