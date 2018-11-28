package cli

type initializer interface {
	init(*App)
}

var _ initializer = &CommonOptions{}

// CommonOptions contains common flags for all commands
type CommonOptions struct {
	LogOptions
}

func (o *CommonOptions) init(app *App) {
	o.LogOptions.init(app)
}
