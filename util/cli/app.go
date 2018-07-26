package cli

import (
	"os"

	flags "github.com/jessevdk/go-flags"
)

// App defines the CLI application that will be run.
type App struct {
	*flags.Parser

	Name string
}

// New creates a new App, including default values.
func New(name string) *App {
	app := &App{Name: name}
	parser := flags.NewParser(nil, flags.Default)
	parser.CommandHandler = func(command flags.Commander, args []string) error {
		if v, ok := command.(initializer); ok {
			v.init(app)
		}
		return command.Execute(args)
	}
	app.Parser = parser
	return app
}

// RunMain parses arguments and runs commands
func (app *App) RunMain() {
	if _, err := app.Parse(); err != nil {
		if err, ok := err.(*flags.Error); ok {
			if err.Type == flags.ErrHelp {
				os.Exit(0)
			}

			app.WriteHelp(os.Stdout)
		}

		os.Exit(1)
	}
}
