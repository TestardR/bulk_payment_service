package sqlite

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type Client struct {
	db     *sql.DB
	config Config
}

func NewClient(config Config) (*Client, error) {
	dsn := buildDSN(config)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err = db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	return &Client{
		db:     db,
		config: config,
	}, nil
}

func buildDSN(config Config) string {
	dsn := fmt.Sprintf("file:%s?", config.DatabasePath)

	dsn += fmt.Sprintf("_busy_timeout=%d", int(config.BusyTimeout.Milliseconds()))

	// Use IMMEDIATE transactions by default to acquire reserved lock immediately
	// This prevents race conditions while still allowing concurrent reads
	dsn += "&_txlock=immediate"

	if config.EnableWAL {
		dsn += "&_journal_mode=WAL"
	}

	return dsn
}

func (c *Client) DB() *sql.DB {
	return c.db
}

func (c *Client) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}
