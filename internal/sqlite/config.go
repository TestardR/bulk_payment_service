package sqlite

import (
	"time"
)

type Config struct {
	DatabasePath    string        `envconfig:"DATABASE_PATH" default:"payments.db"`
	MaxOpenConns    int           `envconfig:"MAX_OPEN_CONNS" default:"25"`
	MaxIdleConns    int           `envconfig:"MAX_IDLE_CONNS" default:"5"`
	ConnMaxLifetime time.Duration `envconfig:"CONN_MAX_LIFETIME" default:"5m"`
	ConnMaxIdleTime time.Duration `envconfig:"CONN_MAX_IDLE_TIME" default:"1m"`
	BusyTimeout     time.Duration `envconfig:"BUSY_TIMEOUT" default:"30s"` // Time to wait for lock acquisition
	EnableWAL       bool          `envconfig:"ENABLE_WAL" default:"true"`  // Allows concurrent reads while writing
}
