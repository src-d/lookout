package main

import (
	"github.com/src-d/lookout/store"
	"github.com/src-d/lookout/util/cli"

	"github.com/golang-migrate/migrate"
	gocli "gopkg.in/src-d/go-cli.v0"
	log "gopkg.in/src-d/go-log.v1"
)

func init() {
	app.AddCommand(&MigrateCommand{})
}

type MigrateCommand struct {
	gocli.PlainCommand `name:"migrate" short-description:"performs a DB migration up to the latest version" long-description:"Performs a DB migration up to the latest version"`
	cli.LogOptions
	cli.DBOptions
}

func (c *MigrateCommand) Execute(args []string) error {
	m, err := store.NewMigrateDSN(c.DB)
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
