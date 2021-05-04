package gcp

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/arehmandev/gcp-nuke/config"
	"github.com/arehmandev/gcp-nuke/helpers"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/syncmap"
	"google.golang.org/api/compute/v1"
)

// ComputeFirewalls -
type ComputeFirewalls struct {
	serviceClient *compute.Service
	base          ResourceBase
	resourceMap   syncmap.Map
}

func init() {
	computeService, err := compute.NewService(Ctx)
	if err != nil {
		log.Fatal(err)
	}
	computeResource := ComputeFirewalls{
		serviceClient: computeService,
	}
	register(&computeResource)
}

// Name - Name of the resourceLister for ComputeFirewalls
func (c *ComputeFirewalls) Name() string {
	return "ComputeFirewalls"
}

// ToSlice - Name of the resourceLister for ComputeFirewalls
func (c *ComputeFirewalls) ToSlice() (slice []string) {
	return helpers.SortedSyncMapKeys(&c.resourceMap)

}

// Setup - populates the struct
func (c *ComputeFirewalls) Setup(config config.Config) {
	c.base.config = config

}

// List - Returns a list of all ComputeFirewalls
func (c *ComputeFirewalls) List(refreshCache bool) []string {
	if !refreshCache {
		return c.ToSlice()
	}
	// Refresh resource map
	c.resourceMap = sync.Map{}

	firewallListCall := c.serviceClient.Firewalls.List(c.base.config.Project)
	firewallList, err := firewallListCall.Do()
	if err != nil {
		log.Fatal(err)
	}

	for _, firewall := range firewallList.Items {

		c.resourceMap.Store(firewall.Name, firewall.Name)
	}
	return c.ToSlice()
}

// Dependencies - Returns a List of resource names to check for
func (c *ComputeFirewalls) Dependencies() []string {
	a := ComputeInstanceGroupsRegion{}
	b := ComputeInstanceGroupsZone{}
	cl := ContainerGKEClusters{}
	return []string{a.Name(), b.Name(), cl.Name()}
}

// Remove -
func (c *ComputeFirewalls) Remove() error {

	// Removal logic
	errs, _ := errgroup.WithContext(c.base.config.Context)

	c.resourceMap.Range(func(key, value interface{}) bool {
		firewallID := key.(string)
		zone := value.(DefaultResourceProperties).zone

		// Parallel firewall deletion
		errs.Go(func() error {
			deleteCall := c.serviceClient.Firewalls.Delete(c.base.config.Project, firewallID)
			operation, err := deleteCall.Do()
			if err != nil {
				return err
			}
			var opStatus string
			seconds := 0
			for opStatus != "DONE" {
				log.Printf("[Info] Resource currently being deleted %v [type: %v project: %v zone: %v] (%v seconds)", firewallID, c.Name(), c.base.config.Project, zone, seconds)

				operationCall := c.serviceClient.ZoneOperations.Get(c.base.config.Project, zone, operation.Name)
				checkOpp, err := operationCall.Do()
				if err != nil {
					return err
				}
				opStatus = checkOpp.Status

				time.Sleep(time.Duration(c.base.config.PollTime) * time.Second)
				seconds += c.base.config.PollTime
				if seconds > c.base.config.Timeout {
					return fmt.Errorf("[Error] Resource deletion timed out for %v [type: %v project: %v zone: %v] (%v seconds)", firewallID, c.Name(), c.base.config.Project, zone, c.base.config.Timeout)
				}
			}
			c.resourceMap.Delete(firewallID)

			log.Printf("[Info] Resource deleted %v [type: %v project: %v zone: %v] (%v seconds)", firewallID, c.Name(), c.base.config.Project, zone, seconds)
			return nil
		})
		return true
	})
	// Wait for all deletions to complete, and return the first non nil error
	err := errs.Wait()
	return err
}
