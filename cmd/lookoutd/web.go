package main

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/src-d/lookout/store"
	"github.com/src-d/lookout/store/models"

	"github.com/src-d/lookout/util/cli"
	"github.com/src-d/lookout/web"
	gocli "gopkg.in/src-d/go-cli.v0"
	log "gopkg.in/src-d/go-log.v1"
	yaml "gopkg.in/yaml.v2"
)

func init() {
	app.AddCommand(&WebCommand{})
}

type WebCommand struct {
	gocli.PlainCommand `name:"web" short-description:"run web server" long-description:"Starts web server for lookoutd"`
	lookoutdBaseCommand
	cli.DBOptions

	Host       string `long:"host" env:"LOOKOUT_WEB_HOST" default:"0.0.0.0" description:"IP address to bind the HTTP server"`
	Port       int    `long:"port" env:"LOOKOUT_WEB_PORT" default:"8080" description:"Port to bind the HTTP server"`
	ServerURL  string `long:"server" env:"LOOKOUT_SERVER_URL" description:"URL used to access the web server in the form 'HOSTNAME[:PORT]'. Leave it unset to allow connections from any proxy or public address"`
	FooterHTML string `long:"footer" env:"LOOKOUT_FOOTER_HTML" description:"Allows to add any custom html to the page footer. It must be a string encoded in base64. Use it, for example, to add your analytics tracking code snippet"`
}

type webConfig struct {
	Providers struct {
		Github struct {
			PrivateKey   string `yaml:"private_key"`
			AppID        int    `yaml:"app_id"`
			ClientID     string `yaml:"client_id"`
			ClientSecret string `yaml:"client_secret"`
		}
	}
	Web struct {
		SigningKey string `yaml:"signing_key"`
	}
}

func (c *WebCommand) Execute(args []string) error {
	var conf webConfig
	configData, err := ioutil.ReadFile(c.ConfigFile)
	if err != nil {
		return fmt.Errorf("Can't open configuration file: %s", err)
	}

	if err := yaml.Unmarshal([]byte(configData), &conf); err != nil {
		return fmt.Errorf("Can't parse configuration file: %s", err)
	}

	ghConfg := conf.Providers.Github
	if ghConfg.PrivateKey == "" {
		return fmt.Errorf("Missing field in configuration file: provider github private_key is required")
	}
	if ghConfg.AppID == 0 {
		return fmt.Errorf("Missing field in configuration file: provider github app_id is required")
	}
	if ghConfg.ClientID == "" {
		return fmt.Errorf("Missing field in configuration file: provider github client_id is required")
	}
	if ghConfg.ClientSecret == "" {
		return fmt.Errorf("Missing field in configuration file: provider github client_secret is required")
	}
	if conf.Web.SigningKey == "" {
		return fmt.Errorf("Missing field in configuration file: web signing_key is required")
	}

	db, err := c.InitDB()
	if err != nil {
		return fmt.Errorf("Can't connect to the DB: %s", err)
	}

	auth := web.NewAuth(ghConfg.ClientID, ghConfg.ClientSecret, conf.Web.SigningKey)

	orgStore := models.NewOrganizationStore(db)
	orgOp := store.NewDBOrganizationOperator(orgStore)
	gh := web.GitHub{
		AppID:          ghConfg.AppID,
		PrivateKey:     ghConfg.PrivateKey,
		OrganizationOp: orgOp,
	}

	static := web.NewStatic("/build/public", c.ServerURL, c.FooterHTML)
	server := web.NewHTTPServer(auth, &gh, static)
	addr := fmt.Sprintf("%s:%d", c.Host, c.Port)

	log.Infof("Starting http server on %s", addr)
	return http.ListenAndServe(addr, server)
}
