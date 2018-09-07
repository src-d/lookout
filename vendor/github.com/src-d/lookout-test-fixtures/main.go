package fixtures

import (
	"gopkg.in/src-d/lookout-sdk.v0/pb"
)

var fixtures = Fixtures{
	{
		Name: "new-go-file-too-long-line",
		CommitRevision: pb.CommitRevision{
			Base: pb.ReferencePointer{
				InternalRepositoryURL: "https://github.com/src-d/lookout-test-fixtures",
				ReferenceName:         "refs/heads/master",
				Hash:                  "6a92946068897d0a6f6ffa6457f889163dcc51b5",
			},
			Head: pb.ReferencePointer{
				InternalRepositoryURL: "https://github.com/src-d/lookout-test-fixtures",
				ReferenceName:         "refs/heads/new_go_file",
				Hash:                  "f47f238d1734881b17e6000cb9fd01248073c900",
			},
		},
		PR: PullRequest{
			URL: "https://github.com/src-d/lookout-test-fixtures/pull/14",
		},
	},
	{
		Name: "file-delete",
		CommitRevision: pb.CommitRevision{
			Base: pb.ReferencePointer{
				InternalRepositoryURL: "https://github.com/src-d/lookout-test-fixtures",
				ReferenceName:         "refs/heads/master",
				Hash:                  "c6e7417ff3b84560f44fe940876494f58e6d68f1",
			},
			Head: pb.ReferencePointer{
				InternalRepositoryURL: "https://github.com/src-d/lookout-test-fixtures",
				ReferenceName:         "refs/heads/remove_file",
				Hash:                  "d0b400aab40eff88066fbe9e27f8aa64f5530538",
			},
		},
		PR: PullRequest{
			URL: "https://github.com/src-d/lookout-test-fixtures/pull/2",
		},
	},
	{
		Name: "file-rename",
		CommitRevision: pb.CommitRevision{
			Base: pb.ReferencePointer{
				InternalRepositoryURL: "https://github.com/src-d/lookout-test-fixtures",
				ReferenceName:         "refs/heads/master",
				Hash:                  "6a92946068897d0a6f6ffa6457f889163dcc51b5",
			},
			Head: pb.ReferencePointer{
				InternalRepositoryURL: "https://github.com/src-d/lookout-test-fixtures",
				ReferenceName:         "refs/heads/rename_file",
				Hash:                  "6221d2fe0bc2148debfa8d3c8c92b8c15451920d",
			},
		},
		PR: PullRequest{
			URL: "https://github.com/src-d/lookout-test-fixtures/pull/3",
		},
	},
	{
		Name: "bblfsh-unknown-language",
		CommitRevision: pb.CommitRevision{
			Base: pb.ReferencePointer{
				InternalRepositoryURL: "https://github.com/src-d/lookout-test-fixtures",
				ReferenceName:         "refs/heads/master",
				Hash:                  "6a92946068897d0a6f6ffa6457f889163dcc51b5",
			},
			Head: pb.ReferencePointer{
				InternalRepositoryURL: "https://github.com/src-d/lookout-test-fixtures",
				ReferenceName:         "refs/heads/bblfsh_unknown_lang",
				Hash:                  "8bc4f2b51be093d261a5ea58e390bc94c1ca3401",
			},
		},
		PR: PullRequest{
			URL: "https://github.com/src-d/lookout-test-fixtures/pull/4",
		},
	},
	{
		Name: "not-ff-merge",
		CommitRevision: pb.CommitRevision{
			Base: pb.ReferencePointer{
				InternalRepositoryURL: "https://github.com/src-d/lookout-test-fixtures",
				ReferenceName:         "refs/heads/i197-base",
				Hash:                  "1f5664bfe6a04a33b6de17a4df0b051d7a43b918",
			},
			Head: pb.ReferencePointer{
				InternalRepositoryURL: "https://github.com/src-d/lookout-test-fixtures",
				ReferenceName:         "refs/heads/i197-head",
				Hash:                  "5fe468b62112e69bae390051e990271f7b1cc294",
			},
		},
		PR: PullRequest{
			URL: "https://github.com/src-d/lookout-test-fixtures/pull/1",
		},
	},
	{
		Name: "merge-conflict",
		CommitRevision: pb.CommitRevision{
			Base: pb.ReferencePointer{
				InternalRepositoryURL: "https://github.com/src-d/lookout-test-fixtures",
				ReferenceName:         "refs/heads/master",
				Hash:                  "6a92946068897d0a6f6ffa6457f889163dcc51b5",
			},
			Head: pb.ReferencePointer{
				InternalRepositoryURL: "https://github.com/src-d/lookout-test-fixtures",
				ReferenceName:         "refs/heads/merge_conflict",
				Hash:                  "72745c8f112cff929927dd12a3decf45239c30e2",
			},
		},
		PR: PullRequest{
			URL: "https://github.com/src-d/lookout-test-fixtures/pull/7",
		},
	},
	{
		Name: "no-content-changes",
		CommitRevision: pb.CommitRevision{
			Base: pb.ReferencePointer{
				InternalRepositoryURL: "https://github.com/src-d/lookout-test-fixtures",
				ReferenceName:         "refs/heads/master",
				Hash:                  "6a92946068897d0a6f6ffa6457f889163dcc51b5",
			},
			Head: pb.ReferencePointer{
				InternalRepositoryURL: "https://github.com/src-d/lookout-test-fixtures",
				ReferenceName:         "refs/heads/no_changes",
				Hash:                  "db7473f2d158b05550c7424f249840ad30f43dc2",
			},
		},
		PR: PullRequest{
			URL: "https://github.com/src-d/lookout-test-fixtures/pull/8",
		},
	},
	{
		Name: "config-empty",
		CommitRevision: pb.CommitRevision{
			Base: pb.ReferencePointer{
				InternalRepositoryURL: "https://github.com/src-d/lookout-test-fixtures",
				ReferenceName:         "refs/heads/master",
				Hash:                  "6a92946068897d0a6f6ffa6457f889163dcc51b5",
			},
			Head: pb.ReferencePointer{
				InternalRepositoryURL: "https://github.com/src-d/lookout-test-fixtures",
				ReferenceName:         "refs/heads/config_empty",
				Hash:                  "8a3e52d451483fe50fefe801f5b7715fce82d4f1",
			},
		},
		PR: PullRequest{
			URL: "https://github.com/src-d/lookout-test-fixtures/pull/10",
		},
	},
	{
		Name: "config-unknown-analyzer",
		CommitRevision: pb.CommitRevision{
			Base: pb.ReferencePointer{
				InternalRepositoryURL: "https://github.com/src-d/lookout-test-fixtures",
				ReferenceName:         "refs/heads/master",
				Hash:                  "6a92946068897d0a6f6ffa6457f889163dcc51b5",
			},
			Head: pb.ReferencePointer{
				InternalRepositoryURL: "https://github.com/src-d/lookout-test-fixtures",
				ReferenceName:         "refs/heads/config_unknown_analyzer",
				Hash:                  "f125c7577f200c4498df840f518c96dc68396266",
			},
		},
		PR: PullRequest{
			URL: "https://github.com/src-d/lookout-test-fixtures/pull/11",
		},
	},
	{
		Name: "submodule-add",
		CommitRevision: pb.CommitRevision{
			Base: pb.ReferencePointer{
				InternalRepositoryURL: "https://github.com/src-d/lookout-test-fixtures",
				ReferenceName:         "refs/heads/master",
				Hash:                  "6a92946068897d0a6f6ffa6457f889163dcc51b5",
			},
			Head: pb.ReferencePointer{
				InternalRepositoryURL: "https://github.com/src-d/lookout-test-fixtures",
				ReferenceName:         "refs/heads/submodule_add",
				Hash:                  "689355891e2ce74e37439574bf1b0af46c9428ed",
			},
		},
		PR: PullRequest{
			URL: "https://github.com/src-d/lookout-test-fixtures/pull/12",
		},
	},
	{
		Name: "submodule-update",
		CommitRevision: pb.CommitRevision{
			Base: pb.ReferencePointer{
				InternalRepositoryURL: "https://github.com/src-d/lookout-test-fixtures",
				ReferenceName:         "refs/heads/master",
				Hash:                  "6a92946068897d0a6f6ffa6457f889163dcc51b5",
			},
			Head: pb.ReferencePointer{
				InternalRepositoryURL: "https://github.com/src-d/lookout-test-fixtures",
				ReferenceName:         "refs/heads/submodule_update",
				Hash:                  "88560a7f9a4e54f7a8d83313dad6be31cb986d2f",
			},
		},
		PR: PullRequest{
			URL: "https://github.com/src-d/lookout-test-fixtures/pull/13",
		},
	},
}

// PullRequest is a struct with information about pull request
type PullRequest struct {
	URL string
	// add recorded responses here later?
}

// Fixture is struct for a test case
type Fixture struct {
	Name           string
	CommitRevision pb.CommitRevision
	PR             PullRequest
}

// Fixtures is a list of fixtures
type Fixtures []*Fixture

// GetByName returns fixture by name
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
