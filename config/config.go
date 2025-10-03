package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"

	"qonto/internal/http"
	"qonto/internal/sqlite"
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
