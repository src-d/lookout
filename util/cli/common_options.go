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
	GrpcMaxMsgSize int    `long:"grpc-max-message-size" default:"100" env:"LOOKOUT_GRPC_MAX_MSG_SIZE" description:"max. message size to send/receive to/from clients (in MB)"`
	DataServer     string `long:"data-server" default:"ipv4://localhost:10301" env:"LOOKOUT_DATA_SERVER" description:"gRPC URL to bind the data server to"`
	Bblfshd        string `long:"bblfshd" default:"ipv4://localhost:9432" env:"LOOKOUT_BBLFSHD" description:"gRPC URL of the Bblfshd server"`
}

func (o *CommonOptions) init(app *App) {
	o.LogOptions.init(app)
	grpchelper.SetMaxMessageSize(o.GrpcMaxMsgSize)
}
