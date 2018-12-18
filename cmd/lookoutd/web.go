package main

import (
	"fmt"
	"net/http"

	"github.com/src-d/lookout/util/cli"
	"github.com/src-d/lookout/web"
	gocli "gopkg.in/src-d/go-cli.v0"
	log "gopkg.in/src-d/go-log.v1"
)

func init() {
	app.AddCommand(&WebCommand{})
}

type WebCommand struct {
	gocli.PlainCommand `name:"web" short-description:"run web server" long-description:"Starts web server for lookoutd"`
	Host               string `long:"host" env:"LOOKOUT_WEB_HOST" default:"0.0.0.0" description:"IP address to bind the HTTP server"`
	Port               int    `long:"port" env:"LOOKOUT_WEB_PORT" default:"8080" description:"Port to bind the HTTP server"`
	ServerURL          string `long:"server" env:"LOOKOUT_SERVER_URL" description:"URL used to access the web server in the form 'HOSTNAME[:PORT]'. Leave it unset to allow connections from any proxy or public address"`
	FooterHTML         string `long:"footer" env:"LOOKOUT_FOOTER_HTML" description:"Allows to add any custom html to the page footer. It must be a string encoded in base64. Use it, for example, to add your analytics tracking code snippet"`

	cli.LogOptions
}

func (c *WebCommand) Execute(args []string) error {
	static := web.NewStatic("build/public", c.ServerURL, c.FooterHTML)
	server := web.NewHTTPServer(static)
	addr := fmt.Sprintf("%s:%d", c.Host, c.Port)

	log.Infof("Starting http server on %s", addr)
	return http.ListenAndServe(addr, server)
}
