package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"

	"payment/internal/http"
	"payment/internal/sqlite"
)

type Config struct {
	LogLevel int `envconfig:"LOG_LEVEL" default:"-4"`
	Database sqlite.Config
	HTTP     http.Config
}

func Load() (Config, error) {
	var config Config
	err := envconfig.Process("", &config)
	if err != nil {
		return Config{}, fmt.Errorf("failed to process config: %w", err)
	}

	return config, nil
}
