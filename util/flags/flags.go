package flags

import (
	"os"

	flags "github.com/jessevdk/go-flags"
	"github.com/src-d/lookout/util/grpchelper"
)

// CommonOptions contains common flags for all commands
type CommonOptions struct {
	GrpcMaxMsgSize int `long:"grpc-max-message-size" default:"100" env:"LOOKOUT_GRPC_MAX_MSG_SIZE" description:"max. message size to send/receive to/from clients (in MB)"`
}

// GetGrpcMaxMsgSize implements GrpcMaxMsgSizer interface
func (o *CommonOptions) GetGrpcMaxMsgSize() int {
	return o.GrpcMaxMsgSize
}

// GrpcMaxMsgSizer is used to get gRPC maximum message size
type GrpcMaxMsgSizer interface {
	GetGrpcMaxMsgSize() int
}

// NewParser returns new flags.Parser
func NewParser() *flags.Parser {
	parser := flags.NewParser(nil, flags.Default)
	parser.CommandHandler = func(command flags.Commander, args []string) error {
		if s, ok := command.(GrpcMaxMsgSizer); ok {
			grpchelper.SetMaxMessageSize(s.GetGrpcMaxMsgSize())
		}
		return command.Execute(args)
	}
	return parser
}

// RunMain parses arguments and runs commands
func RunMain(parser *flags.Parser) {
	if _, err := parser.Parse(); err != nil {
		if err, ok := err.(*flags.Error); ok {
			if err.Type == flags.ErrHelp {
				os.Exit(0)
			}

			parser.WriteHelp(os.Stdout)
		}

		os.Exit(1)
	}
}
