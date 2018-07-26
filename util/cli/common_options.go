package cli

import (
	"github.com/src-d/lookout/util/grpchelper"
)

type initializer interface {
	init(*App)
}

// CommonOptions contains common flags for all commands
type CommonOptions struct {
	LogOptions
	GrpcMaxMsgSize int `long:"grpc-max-message-size" default:"100" env:"LOOKOUT_GRPC_MAX_MSG_SIZE" description:"max. message size to send/receive to/from clients (in MB)"`
}

func (o *CommonOptions) init(app *App) {
	o.LogOptions.init(app)
	grpchelper.SetMaxMessageSize(o.GrpcMaxMsgSize)
}
