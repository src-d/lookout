package mock

import "github.com/src-d/lookout"

type SliceChangeScanner struct {
	Changes    []*lookout.Change
	Error      error
	ChangeTick chan struct{}
	val        *lookout.Change
}

func (s *SliceChangeScanner) Next() bool {
	if s.Error != nil {
		return false
	}

	if len(s.Changes) == 0 {
		s.val = nil
		return false
	}

	s.val, s.Changes = s.Changes[0], s.Changes[1:]
	return true
}

func (s *SliceChangeScanner) Err() error {
	return s.Error
}

func (s *SliceChangeScanner) Change() *lookout.Change {
	if s.ChangeTick != nil {
		<-s.ChangeTick
	}

	return s.val
}

func (s *SliceChangeScanner) Close() error {
	return nil
}

type SliceFileScanner struct {
	Files    []*lookout.File
	Error    error
	FileTick chan struct{}
	val      *lookout.File
}

func (s *SliceFileScanner) Next() bool {
	if s.Error != nil {
		return false
	}

	if len(s.Files) == 0 {
		s.val = nil
		return false
	}

	s.val, s.Files = s.Files[0], s.Files[1:]
	return true
}

func (s *SliceFileScanner) Err() error {
	return s.Error
}

func (s *SliceFileScanner) File() *lookout.File {
	if s.FileTick != nil {
		<-s.FileTick
	}

	return s.val
}

func (s *SliceFileScanner) Close() error {
	return nil
}
