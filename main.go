package main

import (
	"log"
	"os"

	"github.com/arehmandev/gcp-nuke/config"
	"github.com/arehmandev/gcp-nuke/gcp"
)

func main() {

	// Behaviour to delete one project at a time - will be made into loop later
	project := os.Getenv("GCP_PROJECT_ID")
	if project == "" {
		log.Fatalln("GCP_PROJECT_ID environment variable not set")
	}
	config := config.Config{
		Project:  project,
		DryRun:   false,
		Timeout:  400,
		PollTime: 10,
		Context:  gcp.Ctx,
		Zones:    gcp.GetZones(gcp.Ctx, project),
		Regions:  gcp.GetRegions(gcp.Ctx, project),
	}
	log.Printf("[Info] Timeout %v seconds. Polltime %v seconds", config.Timeout, config.PollTime)
	gcp.RemoveProject(config)
}
