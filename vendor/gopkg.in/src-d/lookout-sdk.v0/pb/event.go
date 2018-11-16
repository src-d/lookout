package pb

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
)

// EventID unique hash id for an event
type EventID [20]byte

// ComputeEventID compute the hash for a given list of strings.
func ComputeEventID(content ...string) EventID {
	var id EventID
	h := sha1.New()
	h.Write([]byte(strings.Join(content, "|")))
	copy(id[:], h.Sum(nil))
	return id
}

// IsZero checks if EventID is empty
func (h EventID) IsZero() bool {
	var empty EventID
	return h == empty
}

func (h EventID) String() string {
	return hex.EncodeToString(h[:])
}

// EventType supported event types
type EventType int

const (
	_ EventType = iota
	// PushEventType is an event type when a repository branch gets updated
	PushEventType
	// ReviewEventType is an event type for proposed changes like pull request
	ReviewEventType
)

// ID honors the Event interface.
func (e *ReviewEvent) ID() EventID {
	return ComputeEventID(e.Provider, e.InternalID, e.Head.Hash)
}

// Type honors the Event interface.
func (e *ReviewEvent) Type() EventType {
	return ReviewEventType
}

// Revision honors the Event interface.
func (e *ReviewEvent) Revision() *CommitRevision {
	return &e.CommitRevision
}

// Validate honors the Event interface.
func (e *ReviewEvent) Validate() error {
	var zeroVal ReviewEvent
	if reflect.DeepEqual(*e, zeroVal) {
		return errors.New("this ReviewEvent event is empty")
	}

	return nil
}

// ID honors the Event interface.
func (e *PushEvent) ID() EventID {
	return ComputeEventID(e.Provider, e.InternalID)
}

// Type honors the Event interface.
func (e *PushEvent) Type() EventType {
	return PushEventType
}

// Revision honors the Event interface.
func (e *PushEvent) Revision() *CommitRevision {
	return &e.CommitRevision
}

// Validate honors the Event interface.
func (e *PushEvent) Validate() error {
	var zeroVal PushEvent
	if reflect.DeepEqual(*e, zeroVal) {
		return errors.New("this PushEvent event is empty")
	}

	return nil
}

// RepositoryInfo contains information about a repository
type RepositoryInfo struct {
	CloneURL string
	Host     string
	FullName string
	Owner    string
	Name     string
}

// list of hosts we allow when parse string into RepositoryInfo
var supportedHosts = map[string]bool{
	"github.com":    true,
	"gitlab.com":    true,
	"bitbucket.org": true,
}

// ParseRepositoryInfo creates RepositoryInfo from a string
func ParseRepositoryInfo(input string) (*RepositoryInfo, error) {
	u, err := url.Parse(input)
	if err != nil {
		return nil, err
	}

	if u.Scheme == "" {
		return ParseRepositoryInfo("https://" + input)
	}

	if u.Scheme == "file" {
		return &RepositoryInfo{
			CloneURL: input,
			FullName: u.Path,
			Name:     filepath.Base(u.Path),
		}, nil
	}

	if u.Scheme != "https" {
		return nil, fmt.Errorf("only https urls are supported")
	}

	if u.Host == "" {
		return nil, fmt.Errorf("host can't be empty")
	}

	if _, ok := supportedHosts[u.Host]; !ok {
		return nil, fmt.Errorf("host %s is not supported", u.Host)
	}

	fullName := strings.TrimSuffix(strings.Trim(u.Path, "/"), ".git")
	parts := strings.Split(fullName, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("unsupported path %s", fullName)
	}

	if !strings.HasSuffix(u.Path, ".git") {
		u.Path = u.Path + ".git"
	}

	return &RepositoryInfo{
		CloneURL: u.String(),
		Host:     u.Host,
		FullName: fullName,
		Owner:    parts[0],
		Name:     parts[1],
	}, nil
}

// Repository returns the RepositoryInfo
func (e *ReferencePointer) Repository() *RepositoryInfo {
	info, _ := ParseRepositoryInfo(e.InternalRepositoryURL)
	return info
}

// Short is a short string representation of a ReferencePointer.
func (e *ReferencePointer) Short() string {
	return fmt.Sprintf(
		"%s@%s",
		e.ReferenceName.Short(),
		e.Hash[0:6],
	)
}
