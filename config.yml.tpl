# list of the analyzers to be called by lookoutd
analyzers:
  - name: Dummy
    addr: ipv4://localhost:9930
    disabled: false
    # feedback: url to link in the comment_footer. For example, to open a new GitHub issue
    # settings: map with custom info that will be sent to the analyzer "as is"

providers:
  github:
    comment_footer: "_{{if .Feedback}}If you have feedback about this comment made by the analyzer {{.Name}}, please, [tell us]({{.Feedback}}){{else}}Comment made by the analyzer {{.Name}}{{end}}._"
    # The minimum watch interval to discover new pull requests and push events
    watch_min_interval: 2s
    # Authorization with GitHub App
    # See https://developer.github.com/apps/building-github-apps/authenticating-with-github-apps/
    # app_id: 1234
    # private_key: ./key.pem
    # installation_sync_interval: 1h
    #
    # GitHub App OAuth credentials
    # client_id:
    # client_secret:

# list of repositories to watch when using authorization with a GitHub token
repositories:
  - url: github.com/_USER_/_REPO_TO_WATCH_
    client:
      # user:
      # token:
      # minInterval: 1m

# web interface configuration, only available if authorizing with GitHub App
web:
  # Secret key to sign JSON Web Tokens
  signing_key:

# These are the default timeout values. A value of 0 means no timeout
timeout:
  # Timeout for an analyzer to reply a NotifyReviewEvent
  analyzer_review: 10m
  # Timeout for an analyzer to reply a NotifyPushEvent
  analyzer_push: 60m
  # Timeout for an HTTP requests to the GitHub API
  github_request: 1m
  # Timeout for Git fetch actions
  git_fetch: 20m
  # Timeout for Bblfsh to reply a Parse request
  bblfsh_parse: 2m
