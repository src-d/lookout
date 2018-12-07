package cli

import (
	"encoding/json"

	gocli "gopkg.in/src-d/go-cli.v0"
	log "gopkg.in/src-d/go-log.v1"
)

// LogOptions defines logging flags. It is meant to be embedded in a
// command struct. It is similar to go-cli LogOptions, but adds the application
// name field to the default logger. It also configures the standard logrus
// logger with the same default values as go-log
type LogOptions struct {
	gocli.LogOptions `group:"Log Options"`
}

// Init implements the go-cli initializer interface
func (c LogOptions) Init(a *gocli.App) error {
	if c.LogFields == "" {
		bytes, err := json.Marshal(log.Fields{"app": a.Parser.Name})
		if err != nil {
			panic(err)
		}
		c.LogFields = string(bytes)
	}

	err := c.LogOptions.Init(a)
	if err != nil {
		return err
	}

	log.DefaultFactory.ApplyToLogrus()

	return nil
}
