package gcp

import (
	"fmt"
	"log"
	"sync"

	bq "cloud.google.com/go/bigquery"
	"github.com/BESTSELLER/gcp-nuke/config"
	"github.com/BESTSELLER/gcp-nuke/helpers"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/syncmap"
	"google.golang.org/api/bigquery/v2"
)

type BigQueryDataset struct {
	serviceClient *bigquery.Service
	base          ResourceBase
	resourceMap   syncmap.Map
	DatasetIDs    []string
}

func init() {

	bigqueryService, err := bigquery.NewService(Ctx)
	if err != nil {
		log.Fatal(err)
	}

	bigqueryResource := BigQueryDataset{
		serviceClient: bigqueryService,
	}
	register(&bigqueryResource)
}

func (c *BigQueryDataset) Name() string {
	return "BigQueryDataset"
}

func (c *BigQueryDataset) ToSlice() (slice []string) {
	return helpers.SortedSyncMapKeys(&c.resourceMap)

}

func (c *BigQueryDataset) Setup(config config.Config) {
	c.base.config = config
}

func (c *BigQueryDataset) List(refreshCache bool) []string {
	if !refreshCache {
		return c.ToSlice()
	}
	// Refresh resource map
	c.resourceMap = sync.Map{}

	datasetList, err := c.serviceClient.Datasets.List(c.base.config.Project).Context(Ctx).Do()
	if err != nil {
		log.Fatal(err)
	}

	for _, dataset := range datasetList.Datasets {

		c.resourceMap.Store(dataset.Id, dataset.DatasetReference.DatasetId)

	}

	return c.ToSlice()
}

// Dependencies - Returns a List of resource names to check for
func (c *BigQueryDataset) Dependencies() []string {
	return []string{}
}

func (c *BigQueryDataset) Remove() error {

	client, err := bq.NewClient(Ctx, c.base.config.Project)
	if err != nil {
		return fmt.Errorf("bigquery.NewClient: %v", err)
	}
	// Removal logic
	errs, _ := errgroup.WithContext(c.base.config.Context)

	c.resourceMap.Range(func(key, value interface{}) bool {
		datasetID := value.(string)

		// Parallel instance deletion
		errs.Go(func() error {
			if err := client.Dataset(datasetID).DeleteWithContents(Ctx); err != nil {
				return fmt.Errorf("delete: %v", err)
			}
			deletedTables := false

			seconds := 0
			for deletedTables {

				log.Printf("[Info] Resource currently being deleted %v [type: %v project: %v ] (%v seconds)", datasetID, c.Name(), c.base.config.Project, seconds)
				tableList, _ := c.serviceClient.Tables.List(c.base.config.Project, datasetID).Context(Ctx).Do()
				if tableList == nil {

					deletedTables = true
				}

				seconds += c.base.config.PollTime
				if seconds > c.base.config.Timeout {
					return fmt.Errorf("[Error] Resource deletion timed out for %v [type: %v project: %v ] (%v seconds)", datasetID, c.Name(), c.base.config.Project, c.base.config.Timeout)
				}
			}

			c.resourceMap.Delete(datasetID)
			log.Printf("[Info] Resource deleted %v [type: %v project: %v] (%v seconds)", datasetID, c.Name(), c.base.config.Project, seconds)
			return nil
		})
		return true
	})
	// Wait for all deletions to complete, and return the first non nil error
	err = errs.Wait()
	return err
}
