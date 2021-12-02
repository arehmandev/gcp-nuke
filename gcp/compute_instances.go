package gcp

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/BESTSELLER/gcp-nuke/config"
	"github.com/BESTSELLER/gcp-nuke/helpers"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/syncmap"
	"google.golang.org/api/compute/v1"
)

// ComputeInstances -
type ComputeInstances struct {
	serviceClient *compute.Service
	base          ResourceBase
	resourceMap   syncmap.Map
}

func init() {
	computeService, err := compute.NewService(Ctx)
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

// ToSlice - Name of the resourceLister for ComputeInstances
func (c *ComputeInstances) ToSlice() (slice []string) {
	return helpers.SortedSyncMapKeys(&c.resourceMap)

}

// Setup - populates the struct
func (c *ComputeInstances) Setup(config config.Config) {
	c.base.config = config

}

// List - Returns a list of all ComputeInstances
func (c *ComputeInstances) List(refreshCache bool) []string {
	if !refreshCache {
		return c.ToSlice()
	}
	// Refresh resource map
	c.resourceMap = sync.Map{}

	for _, zone := range c.base.config.Zones {
		instanceListCall := c.serviceClient.Instances.List(c.base.config.Project, zone)
		instanceList, err := instanceListCall.Do()
		if err != nil {
			log.Fatal(err)
		}

		for _, instance := range instanceList.Items {
			skipInstance := false
			// Skip any managed by instance groups
			for _, item := range instance.Metadata.Items {
				if item.Key == "created-by" && strings.Contains(*item.Value, "/instanceGroupManagers/") {
					skipInstance = true
				}
			}
			if skipInstance {
				continue
			}

			instanceResource := DefaultResourceProperties{
				zone: zone,
			}
			c.resourceMap.Store(instance.Name, instanceResource)
		}
	}
	return c.ToSlice()
}

// Dependencies - Returns a List of resource names to check for
func (c *ComputeInstances) Dependencies() []string {
	return []string{}
}

// Remove -
func (c *ComputeInstances) Remove() error {

	// Removal logic
	errs, _ := errgroup.WithContext(c.base.config.Context)

	c.resourceMap.Range(func(key, value interface{}) bool {
		instanceID := key.(string)
		zone := value.(DefaultResourceProperties).zone

		// Parallel instance deletion
		errs.Go(func() error {
			getInstanceCall := c.serviceClient.Instances.Get(c.base.config.Project, zone, instanceID)
			getOp, err := getInstanceCall.Do()
			if err != nil {
				return err
			}
			for _, disk := range getOp.Disks {
				// Set all attached compute disks to auto delete on instance deletion
				diskSetCall := c.serviceClient.Instances.SetDiskAutoDelete(c.base.config.Project, zone, instanceID, true, disk.DeviceName)
				// Todo - check this op until it completes, most likely not needed, but always nice to be safe
				_, err := diskSetCall.Do()
				if err != nil {
					return err
				}
			}
			deleteCall := c.serviceClient.Instances.Delete(c.base.config.Project, zone, instanceID)
			operation, err := deleteCall.Do()
			if err != nil {
				return err
			}
			var opStatus string
			seconds := 0
			for opStatus != "DONE" {
				log.Printf("[Info] Resource currently being deleted %v [type: %v project: %v zone: %v] (%v seconds)", instanceID, c.Name(), c.base.config.Project, zone, seconds)

				operationCall := c.serviceClient.ZoneOperations.Get(c.base.config.Project, zone, operation.Name)
				checkOpp, err := operationCall.Do()
				if err != nil {
					return err
				}
				opStatus = checkOpp.Status

				time.Sleep(time.Duration(c.base.config.PollTime) * time.Second)
				seconds += c.base.config.PollTime
				if seconds > c.base.config.Timeout {
					return fmt.Errorf("[Error] Resource deletion timed out for %v [type: %v project: %v zone: %v] (%v seconds)", instanceID, c.Name(), c.base.config.Project, zone, c.base.config.Timeout)
				}
			}
			c.resourceMap.Delete(instanceID)

			log.Printf("[Info] Resource deleted %v [type: %v project: %v zone: %v] (%v seconds)", instanceID, c.Name(), c.base.config.Project, zone, seconds)
			return nil
		})

		return true
	})
	// Wait for all deletions to complete, and return the first non nil error
	err := errs.Wait()
	return err
}
