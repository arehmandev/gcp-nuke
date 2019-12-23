package cmd

import (
	"log"
	"os"

	"github.com/arehmandev/gcp-nuke/config"
	"github.com/arehmandev/gcp-nuke/gcp"
	"github.com/urfave/cli/v2"
)

// Command -
func Command() {

	app := &cli.App{
		Version:   "v0.1.0",
		UsageText: "e.g. gcp-nuke --project test-nuke-262510 --dryrun",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "project, p",
				Usage:    "GCP project id to nuke",
				Required: true,
			},
			&cli.BoolFlag{
				Name:  "dryrun, d",
				Usage: "Perform a dryrun instead",
			},
			&cli.IntFlag{
				Name:  "timeout, t",
				Value: 400,
				Usage: "Timeout for removal of a single resource in seconds",
			},
			&cli.IntFlag{
				Name:  "polltime, p",
				Value: 10,
				Usage: "Time for polling resource deletion status in seconds",
			},
		},
		Action: func(c *cli.Context) error {

			// Behaviour to delete all resource in parallel in one project at a time - will be made into loop / concurrenct project nuke if required
			config := config.Config{
				Project:  c.String("project"),
				DryRun:   c.Bool("dryrun"),
				Timeout:  c.Int("timeout"),
				PollTime: c.Int("polltime"),
				Context:  gcp.Ctx,
				Zones:    gcp.GetZones(gcp.Ctx, c.String("project")),
				Regions:  gcp.GetRegions(gcp.Ctx, c.String("project")),
			}
			log.Printf("[Info] Timeout %v seconds. Polltime %v seconds. Dry run :%v", config.Timeout, config.PollTime, config.DryRun)
			gcp.RemoveProject(config)

			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
