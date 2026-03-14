package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file"

	_ "github.com/lib/pq"
)

func NewPostgres(dsn, dbname string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)

	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("driver postgres: %w", err)
	}

	migration, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		dbname, driver,
	)
	if err != nil {
		return nil, fmt.Errorf("migrate instance postgres: %w", err)
	}

	// migrate to the latest version
	err = migration.Up()
	if err != nil && err != migrate.ErrNoChange {
		return nil, fmt.Errorf("migration postgres: %w", err)
	}

	return db, nil
}
