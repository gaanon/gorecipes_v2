package store

import (
	"context"
	"fmt"
	"time"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/gaanon/gorecipes_v2/config" // Adjust import path if needed
)

// NewDBPool creates a new database connection pool.
func NewDBPool(cfg config.DBConfig) (*pgxpool.Pool, error) {
	dbPool, err := pgxpool.New(context.Background(), cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Ping the database to verify the connection.
	// Use a context with a timeout for the ping.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = dbPool.Ping(ctx); err != nil {
		dbPool.Close() // Close the pool if ping fails
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	fmt.Println("Successfully connected to the database!")
	return dbPool, nil
}
