package lookout

import (
	"google.golang.org/grpc"
	"gopkg.in/src-d/lookout-sdk.v0/pb"
)

type EventResponse = pb.EventResponse
type Comment = pb.Comment

type AnalyzerClient = pb.AnalyzerClient
type AnalyzerServer = pb.AnalyzerServer

func RegisterAnalyzerServer(s *grpc.Server, srv AnalyzerServer) {
	pb.RegisterAnalyzerServer(s, srv)
}

func NewAnalyzerClient(conn *grpc.ClientConn) AnalyzerClient {
	return pb.NewAnalyzerClient(conn)
}

// AnalyzerConfig is a configuration of analyzer
type AnalyzerConfig struct {
	Name string
	// Addr is gRPC URL.
	// can be defined only in global config, repository-scoped configuration is ignored
	Addr string
	// Disabled repository-scoped configuration can accept only true value, false value is ignored
	Disabled bool
	// Feedback is a url to be linked after each comment
	Feedback string
	// Settings any configuration for an analyzer
	Settings map[string]interface{}
}

// Analyzer is a struct of analyzer client and config
type Analyzer struct {
	Client AnalyzerClient
	Config AnalyzerConfig
}

// AnalyzerComments contains a group of comments and the config for the
// analyzer that created them
type AnalyzerComments struct {
	Config   AnalyzerConfig
	Comments []*Comment
}
