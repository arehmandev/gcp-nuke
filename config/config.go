package config

import (
	"context"
)

// Config -
type Config struct {
	Project string
	Zones   []string
	Timeout int
	Context context.Context
}
