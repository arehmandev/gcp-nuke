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

// ComputeNetworkPeerings -
type ComputeNetworkPeerings struct {
	serviceClient *compute.Service
	base          ResourceBase
	resourceMap   syncmap.Map
}

func init() {
	computeService, err := compute.NewService(Ctx)
	if err != nil {
		log.Fatal(err)
	}
	computeResource := ComputeNetworkPeerings{
		serviceClient: computeService,
	}
	register(&computeResource)
}

// Name - Name of the resourceLister for ComputeNetworkPeerings
func (c *ComputeNetworkPeerings) Name() string {
	return "ComputeNetworkPeerings"
}

// ToSlice - Name of the resourceLister for ComputeNetworkPeerings
func (c *ComputeNetworkPeerings) ToSlice() (slice []string) {
	return helpers.SortedSyncMapKeys(&c.resourceMap)

}

// Setup - populates the struct
func (c *ComputeNetworkPeerings) Setup(config config.Config) {
	c.base.config = config

}

// List - Returns a list of all ComputeNetworkPeerings
func (c *ComputeNetworkPeerings) List(refreshCache bool) []string {
	if !refreshCache {
		return c.ToSlice()
	}
	// Refresh resource map
	c.resourceMap = sync.Map{}

	networkListCall := c.serviceClient.Networks.List(c.base.config.Project)
	networkList, err := networkListCall.Do()
	if err != nil {
		log.Fatal(err)
	}

	for _, network := range networkList.Items {
		for _, networkPeering := range network.Peerings {
			c.resourceMap.Store(networkPeering.Name, network.Name)
		}
	}
	return c.ToSlice()
}

// Dependencies - Returns a List of resource names to check for
func (c *ComputeNetworkPeerings) Dependencies() []string {
	a := ComputeInstanceGroupsRegion{}
	b := ComputeInstanceGroupsZone{}
	cl := ContainerGKEClusters{}
	return []string{a.Name(), b.Name(), cl.Name()}
}

// Remove -
func (c *ComputeNetworkPeerings) Remove() error {

	// Removal logic
	errs, _ := errgroup.WithContext(c.base.config.Context)

	c.resourceMap.Range(func(key, value interface{}) bool {
		networkPeeringID := key.(string)
		networkID := value.(string)

		// Parallel network deletion
		errs.Go(func() error {

			deleteCall := c.serviceClient.Networks.RemovePeering(c.base.config.Project, networkID, &compute.NetworksRemovePeeringRequest{
				Name: networkPeeringID,
			})
			operation, err := deleteCall.Do()
			if err != nil {
				return err
			}
			var opStatus string
			seconds := 0
			for opStatus != "DONE" {
				log.Printf("[Info] Resource currently being deleted %v [type: %v project: %v] (%v seconds)", networkID, c.Name(), c.base.config.Project, seconds)

				operationCall := c.serviceClient.GlobalOperations.Get(c.base.config.Project, operation.Name)
				checkOpp, err := operationCall.Do()
				if err != nil {
					return err
				}
				opStatus = checkOpp.Status

				time.Sleep(time.Duration(c.base.config.PollTime) * time.Second)
				seconds += c.base.config.PollTime
				if seconds > c.base.config.Timeout {
					return fmt.Errorf("[Error] Resource deletion timed out for %v [type: %v project: %v] (%v seconds)", networkID, c.Name(), c.base.config.Project, c.base.config.Timeout)
				}
			}
			c.resourceMap.Delete(networkID)

			log.Printf("[Info] Resource deleted %v [type: %v project: %v] (%v seconds)", networkID, c.Name(), c.base.config.Project, seconds)
			return nil
		})
		return true
	})
	// Wait for all deletions to complete, and return the first non nil error
	err := errs.Wait()
	return err
}
