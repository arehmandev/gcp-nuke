package gcp

import (
	"fmt"
	"log"
	"sync"

	"github.com/BESTSELLER/gcp-nuke/config"
	"github.com/BESTSELLER/gcp-nuke/helpers"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/syncmap"
	"google.golang.org/api/pubsub/v1"
)

type PubSubTopic struct {
	serviceClient *pubsub.Service
	base          ResourceBase
	resourceMap   syncmap.Map
	TopicIDs      []string
}

func init() {

	pubsubService, err := pubsub.NewService(Ctx)
	if err != nil {
		log.Fatal(err)
	}

	pubsubResource := PubSubTopic{
		serviceClient: pubsubService,
	}
	register(&pubsubResource)
}

func (c *PubSubTopic) Name() string {
	return "PubSubTopic"
}

func (c *PubSubTopic) ToSlice() (slice []string) {
	return helpers.SortedSyncMapKeys(&c.resourceMap)

}

func (c *PubSubTopic) Setup(config config.Config) {
	c.base.config = config
}

func (c *PubSubTopic) List(refreshCache bool) []string {
	if !refreshCache {
		return c.ToSlice()
	}
	// Refresh resource map
	c.resourceMap = sync.Map{}

	topicList, err := c.serviceClient.Projects.Topics.List("projects/" + c.base.config.Project).Context(Ctx).Do()
	if err != nil {
		log.Fatal(err)
	}

	for _, topic := range topicList.Topics {
		c.resourceMap.Store(topic.Name, topic.Name)

	}

	return c.ToSlice()
}

// Dependencies - Returns a List of resource names to check for
func (c *PubSubTopic) Dependencies() []string {
	return []string{}
}

func (c *PubSubTopic) Remove() error {
	// Removal logic
	errs, _ := errgroup.WithContext(c.base.config.Context)

	c.resourceMap.Range(func(key, value interface{}) bool {
		topicID := value.(string)
		fmt.Println(topicID)
		// location := strings.Split(datasetID, "/")[3]
		// Parallel instance deletion
		errs.Go(func() error {

			_, err := c.serviceClient.Projects.Topics.Delete(topicID).Context(Ctx).Do()
			if err != nil {
				return err
			}

			seconds := 0

			log.Printf("[Info] Resource deleted %v [type: %v project: %v] (%v seconds)", topicID, c.Name(), c.base.config.Project, seconds)
			return nil
		})
		return true
	})
	// Wait for all deletions to complete, and return the first non nil error
	err := errs.Wait()
	return err
}
