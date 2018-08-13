package purge

import (
	"context"

	"github.com/src-d/lookout"
)

// Service implements data service interface which purges unneeded fields from response
// currently purges only file content
type Service struct {
	changes lookout.ChangeGetter
	files   lookout.FileGetter
}

var _ lookout.ChangeGetter = &Service{}
var _ lookout.FileGetter = &Service{}

// NewService creates new purge Service
func NewService(changes lookout.ChangeGetter, files lookout.FileGetter) *Service {
	return &Service{
		changes: changes,
		files:   files,
	}
}

// GetChanges returns a ChangeScanner that scans all changes according to the request.
func (s *Service) GetChanges(ctx context.Context, req *lookout.ChangesRequest) (lookout.ChangeScanner, error) {
	wantContents := req.WantContents
	changes, err := s.changes.GetChanges(ctx, req)
	if err != nil {
		return nil, err
	}

	if wantContents == req.WantContents {
		return changes, nil
	}

	return &purgeChangesScanner{
		underlying:    changes,
		purgeContents: !wantContents,
	}, nil
}

// GetFiles returns a FilesScanner that scans all files according to the request.
func (s *Service) GetFiles(ctx context.Context, req *lookout.FilesRequest) (lookout.FileScanner, error) {
	wantContents := req.WantContents
	files, err := s.files.GetFiles(ctx, req)
	if err != nil {
		return nil, err
	}

	if wantContents == req.WantContents {
		return files, nil
	}

	return &purgeFileScanner{
		underlying:    files,
		purgeContents: !wantContents,
	}, nil
}

type purgeChangesScanner struct {
	underlying    lookout.ChangeScanner
	purgeContents bool
	val           *lookout.Change
}

func (s *purgeChangesScanner) Next() bool {
	if !s.underlying.Next() {
		return false
	}

	s.val = s.underlying.Change()

	if s.purgeContents {
		if s.val.Base != nil {
			s.val.Base.Content = nil
		}
		if s.val.Head != nil {
			s.val.Head.Content = nil
		}
	}

	return true
}

func (s *purgeChangesScanner) Err() error {
	return s.underlying.Err()
}

func (s *purgeChangesScanner) Change() *lookout.Change {
	return s.val
}

func (s *purgeChangesScanner) Close() error {
	return s.underlying.Close()
}

type purgeFileScanner struct {
	underlying    lookout.FileScanner
	purgeContents bool
	val           *lookout.File
}

func (s *purgeFileScanner) Next() bool {
	if !s.underlying.Next() {
		return false
	}

	s.val = s.underlying.File()

	if s.purgeContents {
		s.val.Content = nil
	}

	return true
}

func (s *purgeFileScanner) Err() error {
	return s.underlying.Err()
}

func (s *purgeFileScanner) File() *lookout.File {
	return s.val
}

func (s *purgeFileScanner) Close() error {
	return s.underlying.Close()
}
