package fixtures

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/google/go-github/github"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/lookout-sdk.v0/pb"
)

var fixtures = Fixtures{
	{
		Name: "new-go-file-too-long-line",
		URL: PullRequestURL{
			Owner:  "src-d",
			Repo:   "lookout-test-fixtures",
			Number: 14,
		},
		CurrentRevision: 0,
	},
	{
		Name: "file-delete",
		URL: PullRequestURL{
			Owner:  "src-d",
			Repo:   "lookout-test-fixtures",
			Number: 2,
		},
		CurrentRevision: 0,
	},
	{
		Name: "file-rename",
		URL: PullRequestURL{
			Owner:  "src-d",
			Repo:   "lookout-test-fixtures",
			Number: 3,
		},
		CurrentRevision: 0,
	},
	{
		Name: "bblfsh-unknown-language",
		URL: PullRequestURL{
			Owner:  "src-d",
			Repo:   "lookout-test-fixtures",
			Number: 4,
		},
		CurrentRevision: 0,
	},
	{
		Name: "not-ff-merge",
		URL: PullRequestURL{
			Owner:  "src-d",
			Repo:   "lookout-test-fixtures",
			Number: 1,
		},
		CurrentRevision: 0,
	},
	{
		Name: "merge-conflict",
		URL: PullRequestURL{
			Owner:  "src-d",
			Repo:   "lookout-test-fixtures",
			Number: 7,
		},
		CurrentRevision: 0,
	},
	{
		Name: "no-content-changes",
		URL: PullRequestURL{
			Owner:  "src-d",
			Repo:   "lookout-test-fixtures",
			Number: 8,
		},
		CurrentRevision: 0,
	},
	{
		Name: "config-empty",
		URL: PullRequestURL{
			Owner:  "src-d",
			Repo:   "lookout-test-fixtures",
			Number: 10,
		},
		CurrentRevision: 0,
	},
	{
		Name: "config-unknown-analyzer",
		URL: PullRequestURL{
			Owner:  "src-d",
			Repo:   "lookout-test-fixtures",
			Number: 11,
		},
		CurrentRevision: 0,
	},
	{
		Name: "submodule-add",
		URL: PullRequestURL{
			Owner:  "src-d",
			Repo:   "lookout-test-fixtures",
			Number: 12,
		},
		CurrentRevision: 0,
	},
	{
		Name: "submodule-update",
		URL: PullRequestURL{
			Owner:  "src-d",
			Repo:   "lookout-test-fixtures",
			Number: 13,
		},
		CurrentRevision: 0,
	},
	{
		Name: "incremental-pr",
		URL: PullRequestURL{
			Owner:  "src-d",
			Repo:   "lookout-test-fixtures",
			Number: 16,
		},
		CurrentRevision: 1,
	},
	{
		Name: "get-changes-from-outdated-pr",
		URL: PullRequestURL{
			Owner:  "src-d",
			Repo:   "lookout-test-fixtures",
			Number: 20,
		},
		CurrentRevision: 0,
	},
	{
		Name: "pr-from-fork",
		URL: PullRequestURL{
			Owner:  "src-d",
			Repo:   "lookout-test-fixtures",
			Number: 26,
		},
		CurrentRevision: 0,
	},
	{
		Name: "pr-draft",
		URL: PullRequestURL{
			Owner:  "src-d",
			Repo:   "lookout-test-fixtures",
			Number: 29,
		},
		CurrentRevision: 0,
	},
}

// PullRequestURL is a struct with information about a pull request URL:
// github.com/<Owner>/<Repo>/pull/<Number>
type PullRequestURL struct {
	Owner  string
	Repo   string
	Number int
}

// Fixture is struct for a test case
type Fixture struct {
	Name string
	URL  PullRequestURL
	// There can be more than one revision of the fixture, but it should have
	// files for 0..CurrentRevision
	CurrentRevision int
}

// GetRevision returns a copy of this Fixture at a different revision
func (f Fixture) GetRevision(revision int) *Fixture {
	copy := f
	copy.CurrentRevision = revision

	return &copy
}

// GetRevisions returns all this Fixture revisions, ordered from 0 to CurrentRevision
func (f Fixture) GetRevisions() []*Fixture {
	list := make([]*Fixture, f.CurrentRevision+1)
	for v := 0; v <= f.CurrentRevision; v++ {
		list[v] = f.GetRevision(v)
	}

	return list
}

// GetPR returns the fixture saved from the github.Client.PullRequests.Get
// output
func (f Fixture) GetPR() *github.PullRequest {
	filename := f.FilenamePR(f.CurrentRevision)
	data, err := Asset(filename)
	if err != nil {
		panic(fmt.Sprintf("unexpected error while loading %s: %s", filename, err.Error()))
	}

	var pr github.PullRequest
	err = json.Unmarshal(data, &pr)
	if err != nil {
		panic(fmt.Sprintf("unexpected error while unmarshalling %s: %s", filename, err.Error()))
	}

	return &pr
}

// GetCommitsComparison returns the fixture saved from the
// github.Client.Repositories.CompareCommits output
func (f Fixture) GetCommitsComparison() (*github.CommitsComparison, error) {
	filename := f.FilenameCC(f.CurrentRevision)
	data, err := Asset(filename)
	if err != nil {
		return nil, err
	}

	var cc github.CommitsComparison
	err = json.Unmarshal(data, &cc)
	if err != nil {
		return nil, err
	}

	return &cc, nil
}

// GetCommitRevision builds a pb.CommitRevision from this fixture's GetPR()
func (f Fixture) GetCommitRevision() *pb.CommitRevision {
	pr := f.GetPR()

	return &pb.CommitRevision{
		Base: pb.ReferencePointer{
			InternalRepositoryURL: pr.GetBase().GetRepo().GetCloneURL(),
			ReferenceName:         plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", pr.GetBase().GetRef())),
			Hash:                  pr.GetBase().GetSHA(),
		},
		Head: pb.ReferencePointer{
			InternalRepositoryURL: pr.GetHead().GetRepo().GetCloneURL(),
			ReferenceName:         plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", pr.GetHead().GetRef())),
			Hash:                  pr.GetHead().GetSHA(),
		},
	}
}

func (f Fixture) FilenamePR(revision int) string {
	return filepath.FromSlash(fmt.Sprintf("fixtures/%s.%s.pr-%d-v%d.json", f.URL.Owner, f.URL.Repo, f.URL.Number, revision))
}

func (f Fixture) FilenameCC(revision int) string {
	return filepath.FromSlash(fmt.Sprintf("fixtures/%s.%s.pr-%d-v%d-compare.json", f.URL.Owner, f.URL.Repo, f.URL.Number, revision))
}

// Fixtures is a list of fixtures. If a fixture has more than one revision the
// most recent one is used
type Fixtures []*Fixture

// GetByName returns fixture by name. If the fixture has more than one revision
// the most recent one is used
func (g Fixtures) GetByName(name string) *Fixture {
	for _, f := range g {
		if f.Name == name {
			return f
		}
	}

	return nil
}

// GetAll returns all fixtures
func GetAll() []*Fixture {
	return fixtures
}

// GetByName returns fixture by name
func GetByName(name string) *Fixture {
	return fixtures.GetByName(name)
}
