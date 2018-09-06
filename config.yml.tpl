analyzers:
  - name: Dummy
    addr: ipv4://localhost:10302
    # feedback: url to link in the comment_footer. For example, to open a new GitHub issue

providers:
  github:
    comment_footer: "_If you have feedback about this comment, please, [tell us](%s)._"
    # See https://developer.github.com/apps/building-github-apps/authenticating-with-github-apps/
    # app_id: 1234
    # private_key: ./key.pem

repositories:
  - url: github.com/src-d/lookout
    client:
      # user:
      # token:
      # minInterval: 1m
