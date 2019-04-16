# lookout-test-fixtures

To update the fixtures use the scraper utility. It will call the GitHub API to list the PRs defined in `var fixtures` in [`main.go`](./main.go). The output will be dumped as JSON files.

It is not required, but you may want to authenticate using these environment variables:

```bash
export GITHUB_USER=xxx
export GITHUB_TOKEN=yyy
```

The fixtures must be bundled using [go-bindata](https://github.com/jteeuwen/go-bindata).

```bash
go run cmd/scrape/main.go
go-bindata -modtime 1536310226 -pkg fixtures fixtures/
```

To store more than one fixture for a PR, for example to test incremental pushes to a branch, increment the `Fixture.CurrentRevision` number by one. The scraper utility will leave existing files, and create new ones with the `-vN` suffix.
