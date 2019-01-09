package lookout

import (
	"google.golang.org/grpc"
	log "gopkg.in/src-d/go-log.v1"
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

// AnalyzerCommentsGroups list of AnalyzerComments
type AnalyzerCommentsGroups []AnalyzerComments

// CommentsFilterFn is a function that filters comments
type CommentsFilterFn func(*Comment) (skip bool, err error)

// Filter filters comments groups using CommentsFilterFn
func (g AnalyzerCommentsGroups) Filter(fn CommentsFilterFn) ([]AnalyzerComments, error) {
	var result []AnalyzerComments

	for _, group := range g {
		var newComments []*Comment
		for _, c := range group.Comments {
			skip, err := fn(c)
			if err != nil {
				return nil, err
			}

			if !skip {
				newComments = append(newComments, c)
			}
		}

		if len(newComments) > 0 {
			result = append(result, AnalyzerComments{
				Config:   group.Config,
				Comments: newComments,
			})
		}
	}

	return result, nil
}

// Count returns the total number of comments
func (g AnalyzerCommentsGroups) Count() int {
	count := 0
	for _, group := range g {
		count += len(group.Comments)
	}

	return count
}

// Dedup filters duplicated comments
func (g AnalyzerCommentsGroups) Dedup() AnalyzerCommentsGroups {
	var result []AnalyzerComments
	type Key struct {
		File string
		Line int32
		Text string
	}

	for _, group := range g {
		dupComments := 0
		analyzerName := group.Config.Name
		var newComments []*Comment
		uniqueCommentsMap := map[Key]*Comment{}
		for _, comment := range group.Comments {
			key := Key{
				File: comment.File,
				Line: comment.Line,
				Text: comment.Text,
			}
			if _, ok := uniqueCommentsMap[key]; ok {
				dupComments++
				continue
			}

			uniqueCommentsMap[key] = comment
			newComments = append(newComments, comment)
		}

		if len(newComments) > 0 {
			result = append(result, AnalyzerComments{
				Config:   group.Config,
				Comments: newComments,
			})
		}

		if dupComments > 0 {
			log.Warningf("analyzer %s generated %d duplicated comments", analyzerName, dupComments)
		}
	}

	return result
}
