analyzers:
  - name: Dummy
    addr: ipv4://localhost:9930
    # feedback: url to link in the comment_footer. For example, to open a new GitHub issue

providers:
  github:
    comment_footer: '_If you have feedback about this comment, please, [tell us](%s)._'
    # See https://developer.github.com/apps/building-github-apps/authenticating-with-github-apps/
    # app_id: 1234
    # private_key: ./key.pem
    # installation_sync_interval: 1h
    # watch_min_interval: 2s

repositories:
  - url: github.com/src-d/lookout
    client:
      # user:
      # token:
      # minInterval: 1m

# These are the default timeout values. A value of 0 means no timeout
timeout:
  # Timeout for an analyzer response to a NotifyReviewEvent
  analyzer_review: 10m
  # Timeout for an analyzer response to a NotifyPushEvent
  analyzer_push: 60m
  # Timeout http requests to the GitHub API
  github_request: 1m
  # Timeout for Git fetch actions
  git_fetch: 20m
  # Timeout for parse requests to Bblfsh
  bblfsh_parse: 2m
