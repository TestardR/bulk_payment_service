package http

import (
	"time"
)

type Config struct {
	Address string        `envconfig:"HTTP_ADDRESS" default:"localhost:8080"`
	Timeout time.Duration `envconfig:"HTTP_TIMEOUT" default:"10s"`
}
