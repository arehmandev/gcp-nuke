package config

import (
	"context"
)

// Config -
type Config struct {
	Project  string
	Zones    []string
	Regions  []string
	Timeout  int
	PollTime int
	Context  context.Context
	DryRun   bool
}
