package cli

import (
	"database/sql"
	"fmt"

	"github.com/src-d/lookout/store"

	"github.com/golang-migrate/migrate"
	_ "github.com/lib/pq"
	log "gopkg.in/src-d/go-log.v1"
)

// DBOptions contains common flags for commands using the DB
type DBOptions struct {
	DB string `long:"db" default:"postgres://postgres:postgres@localhost:5432/lookout?sslmode=disable" env:"LOOKOUT_DB" description:"connection string to postgres database"`
}

func (c *DBOptions) InitDB() (*sql.DB, error) {
	db, err := sql.Open("postgres", c.DB)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	m, err := store.NewMigrateInstance(db)
	if err != nil {
		return nil, err
	}

	dbVersion, _, err := m.Version()

	// The DB is not initialized
	if err == migrate.ErrNilVersion {
		return nil, fmt.Errorf("the DB is empty, it needs to be initialized with the 'migrate' subcommand")
	}

	if err != nil {
		return nil, err
	}

	maxVersion, err := store.MaxMigrateVersion()
	if err != nil {
		return nil, err
	}

	if dbVersion != maxVersion {
		return nil, fmt.Errorf(
			"database version mismatch. Current version is %v, but this binary needs version %v. "+
				"Use the 'migrate' subcommand to upgrade your database", dbVersion, maxVersion)
	}

	log.With(log.Fields{"db-version": dbVersion}).Debugf("the DB version is up to date")
	log.Infof("connection with the DB established")
	return db, nil
}
