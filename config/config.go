package config

import "context"

// Config -
type Config struct {
	Project string
	Timeout int
	Context context.Context
}
