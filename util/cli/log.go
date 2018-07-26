package cli

import (
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/grpclog"
	"gopkg.in/src-d/go-log.v1"
)

// LogOptions defines logging flags. It is meant to be embedded in a
// command struct.
type LogOptions struct {
	LogLevel       string `long:"log-level" env:"LOG_LEVEL" default:"info" description:"Logging level (info, debug, warning or error)"`
	LogFormat      string `long:"log-format" env:"LOG_FORMAT" description:"log format (text or json), defaults to text on a terminal and json otherwise"`
	LogFields      string `long:"log-fields" env:"LOG_FIELDS" description:"default fields for the logger, specified in json"`
	LogForceFormat bool   `long:"log-force-format" env:"LOG_FORCE_FORMAT" description:"ignore if it is running on a terminal or not"`
	Verbose        bool   `long:"verbose" short:"v" description:"enable verbose logging"`
}

// Init initializes the default logger factory.
func (c *LogOptions) init(app *App) error {
	log.DefaultFactory = &log.LoggerFactory{
		Level:       c.LogLevel,
		Format:      c.LogFormat,
		Fields:      c.LogFields,
		ForceFormat: c.LogForceFormat,
	}
	log.DefaultFactory.ApplyToLogrus()

	log.DefaultLogger = log.New(log.Fields{"app": app.Name})

	// copy current behavior, verbose flag doesn't change level but enables grpc logging
	if c.Verbose {
		grpclog.SetLoggerV2(GrpcLogrus{logrus.StandardLogger()})
	}
	return nil
}

// GrpcLogrus implements grpclog.LoggerV2 interface using logrus logger
type GrpcLogrus struct {
	*logrus.Logger
}

// V reports whether verbosity level l is at least the requested verbose level.
func (lw GrpcLogrus) V(l int) bool {
	// logrus & grpc levels are inverted
	logrusLevel := 4 - l
	return int(lw.Logger.Level) <= logrusLevel
}
