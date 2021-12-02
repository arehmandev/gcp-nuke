package gcp

import (
	"log"
	"strings"
	"sync"

	"github.com/BESTSELLER/gcp-nuke/config"
	"github.com/BESTSELLER/gcp-nuke/helpers"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/syncmap"
	"google.golang.org/api/iam/v1"
)

type IAMServiceAccount struct {
	serviceClient *iam.Service
	base          ResourceBase
	resourceMap   syncmap.Map
	TopicIDs      []string
}

func init() {

	iamService, err := iam.NewService(Ctx)
	if err != nil {
		log.Fatal(err)
	}

	iamResource := IAMServiceAccount{
		serviceClient: iamService,
	}
	register(&iamResource)
}

func (c *IAMServiceAccount) Name() string {
	return "IAMServiceAccount"
}

func (c *IAMServiceAccount) ToSlice() (slice []string) {
	return helpers.SortedSyncMapKeys(&c.resourceMap)

}

func (c *IAMServiceAccount) Setup(config config.Config) {
	c.base.config = config
}

func (c *IAMServiceAccount) List(refreshCache bool) []string {
	if !refreshCache {
		return c.ToSlice()
	}
	// Refresh resource map
	c.resourceMap = sync.Map{}

	serviceAccountList, err := c.serviceClient.Projects.ServiceAccounts.List("projects/" + c.base.config.Project).Do()
	if err != nil {
		log.Fatal(err)
	}

	for _, serviceAccount := range serviceAccountList.Accounts {
		// Will not list / delete default service accounts
		if strings.Contains(serviceAccount.Email, c.base.config.Project) {
			c.resourceMap.Store(serviceAccount.DisplayName, serviceAccount.Email)
		}

	}

	return c.ToSlice()
}

// Dependencies - Returns a List of resource names to check for
func (c *IAMServiceAccount) Dependencies() []string {
	return []string{}
}

func (c *IAMServiceAccount) Remove() error {
	// Removal logic
	errs, _ := errgroup.WithContext(c.base.config.Context)

	c.resourceMap.Range(func(key, value interface{}) bool {
		emailAddress := value.(string)

		// Parallel instance deletion
		errs.Go(func() error {

			_, err := c.serviceClient.Projects.ServiceAccounts.Delete("projects/" + c.base.config.Project + "/serviceAccounts/" + emailAddress).Do()
			if err != nil {
				return err
			}

			seconds := 0

			log.Printf("[Info] Resource deleted %v [type: %v project: %v] (%v seconds)", emailAddress, c.Name(), c.base.config.Project, seconds)
			return nil
		})
		return true
	})
	// Wait for all deletions to complete, and return the first non nil error
	err := errs.Wait()
	return err
}
