package store

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/postgres"
	"github.com/golang-migrate/migrate/source"
)

// NewMigrateDSN returns a new Migrate instance from a database URL
func NewMigrateDSN(dsn string) (*migrate.Migrate, error) {
	source, err := initSource()
	if err != nil {
		return nil, err
	}

	return migrate.NewWithSourceInstance("file", source, dsn)
}

// NewMigrateInstance returns a new Migrate instance from a postgres instance
func NewMigrateInstance(db *sql.DB) (*migrate.Migrate, error) {
	source, err := initSource()
	if err != nil {
		return nil, err
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})

	return migrate.NewWithInstance("file", source, "postgres", driver)
}

func initSource() (source.Driver, error) {
	return WithInstance(FS(false), &Config{Dir: "/store/migrations/"})
}

// MaxMigrateVersion returns the current DB migration file version
func MaxMigrateVersion() (uint, error) {
	var maxVersion uint64

	dir, err := FS(false).Open("/store/migrations/")
	if err != nil {
		return 0, fmt.Errorf("can't open directory with migrations: %s", err)
	}

	files, err := dir.Readdir(-1)
	dir.Close()
	if err != nil {
		return 0, fmt.Errorf("can't read files in migrations directory: %s", err)
	}

	for _, fi := range files {
		name := fi.Name()
		if !strings.HasSuffix(name, "up.sql") {
			continue
		}

		vStr := strings.Split(name, "_")[0]
		vUint, err := strconv.ParseUint(vStr, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("failed to parse migration version for file %s, %s", name, err)
		}

		if vUint > maxVersion {
			maxVersion = vUint
		}
	}

	return uint(maxVersion), nil
}
