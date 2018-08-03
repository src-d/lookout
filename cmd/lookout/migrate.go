package main

import (
	"github.com/src-d/lookout/store"
	"github.com/src-d/lookout/util/cli"

	"github.com/golang-migrate/migrate"
	_ "github.com/golang-migrate/migrate/database/postgres"
	bindata "github.com/golang-migrate/migrate/source/go-bindata"
	log "gopkg.in/src-d/go-log.v1"
)

func init() {
	if _, err := app.AddCommand("migrate", "performs a DB migration up to the latest version", "",
		&MigrateCommand{}); err != nil {
		panic(err)
	}
}

type MigrateCommand struct {
	cli.LogOptions
	cli.DBOptions
}

func (c *MigrateCommand) Execute(args []string) error {
	s := bindata.Resource(store.AssetNames(),
		func(name string) ([]byte, error) {
			return store.Asset(name)
		})

	d, err := bindata.WithInstance(s)
	if err != nil {
		return err
	}

	m, err := migrate.NewWithSourceInstance("go-bindata", d, c.DB)
	if err != nil {
		return err
	}

	err = m.Up()
	switch err {
	case nil:
		log.Infof("The DB was upgraded")
	case migrate.ErrNoChange:
		log.Infof("The DB is up to date")
	default:
		return err
	}

	return nil
}
