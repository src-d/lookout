# Configuring lookout

lookout is configured with the `config.yml` file, you can use the template [`config.yml.tpl`](../config.yml.tpl) to create your own. Use the `lookoutd` option `--config` to set the path to it, or use the default location at `./config.yml`.

The main structure of `config.yml` is:

```yml
providers:
    github:
        # configuration of GitHub provider
repositories:
    # list of repositories to watch and user/token if needed
analyzers:
    # list of named analyzers
```


## Github Provider

The `providers.github` key configures how **lookout** will connect with GitHub. 

```yml
providers:
  github:
    comment_footer: "_If you have feedback about this comment, please, [tell us](%s)._"
    # app_id: 1234
    # private_key: ./key.pem
    # installation_sync_interval: 1h
    # watch_min_interval: 2s
```

`comment_footer` key defines a format-string that will be used for custom messages for every message posted on GitHub; see how to [add a custom message to the posted comments](#custom-footer)

<a id=basic-auth></a>
### Authentication with GitHub

#### Authentication as a GitHub Account

lookout needs to authenticate with GitHub. The easiest method for testing purposes is using a [GitHub personal access token](https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/). The token should have the **repo** scope enabled.

The credential can then be passed to `lookoutd` as:

- user: `--github-user` argument or `GITHUB_USER` environment variable.
- token: `--github-token` argument or `GITHUB_TOKEN` environment variable.

<a id=github-app></a>
#### Authentication as a GitHub App

For a production environment you can use **lookout** as a [GitHub App](https://developer.github.com/apps/about-apps/).

To do so, you must also unset any environment variable or option for the GitHub username and token authentication.

You need to create a GitHub App following the [documentation about creating a GitHub app](https://developer.github.com/apps/building-github-apps/creating-a-github-app/). The permissions that must be set are:

- Repository metadata: Read-only
- Pull requests: Read & write
- Single file: Read-only
- Commit statuses: Read & write

Download a private key following the [documentation about authenticating with GitHub Apps](https://developer.github.com/apps/building-github-apps/authenticating-with-github-apps/) and set the following fields in your `config.yml` file:

```yml
providers:
  github:
    app_id: 1234
    private_key: ./key.pem
    installation_sync_interval: 1h
    watch_min_interval: 10s
```

When the GitHub App authentication method is used, the repositories to analyze are retrieved automatically from the GitHub installations, so `repositories` list from `config.yml` is ignored.

The update interval to discover new installations and repositories is defined by `installation_sync_interval`.

The minimum watch interval to discover new pull requests and push events is defined by `watch_min_interval`.

## Repositories

The list of repositories to be watched by **lookout** is defined by the `repositories` key.

The user and token to be used for the Github authentication can be defined per repository; if you do so, it will override the globally passed user and token.

```yml
repositories:
  - url: github.com/<user>/<repo1>
  - url: github.com/<user>/<repo2>
    client:
      # user: github-user
      # token: github-user-token
      # minInterval: 1m
```

If you're using [Authentication as a GitHub App](#github-app), the list of repositories to be watched will be taken from the GitHub installations.

## Analyzers

Each analyzer to be requested by **lookout** should be defined under `analyzers` key.

```yml
analyzers:
  - name: Example name # required, unique name of the analyzer
    addr: ipv4://localhost:9930 # required, gRPC address
    disabled: false # optional, false by default
    feedback: http://example.com/analyzer # url to link in the comment_footer
    settings: # optional, this field is sent to analyzer "as is"
        threshold: 0.8
```

`feedback` key contains the URL used in the custom footer added to any message posted on GitHub; see how to [add a custom message to the posted comments](#custom-footer)

<a id=custom-footer></a>
### Add a Custom Message to the Posted Comments

You can configure **lookout** to add a custom message to every comment that each analyzer returns. This custom message will be created following the rule:
```
sprinf(providers.github.comment_footer, feedback)
```
If any of those two keys are not defined, the custom message won't be added.

Example:
```text
"_If you have feedback about this comment, please, [tell us](%s)._"
```

### Customize an Analyzer from the Repository

It's possible to customize the Analyzers configuration for each repository. To do that you only need to place a `.lookout.yml` file at the root directory of that repository.

Example:
```yml
analyzers:
  - name: Example name
    disabled: true
    settings:
        threshold: 0.9
        mode: confident
```

The analyzer to configure must be identified with the same `name` in the `.lookout.yml` config file as in **lookout** server configuration, otherwise, it will be ignored.

The repository can disable any analyzer, but it cannot define new analyzers nor enable those that are disabled in the **lookout** server.

The `settings` for each analyzer in the `.lookout.yml` config file will be merged with the **lookout** configuration following these rules:

- Objects are deep merged
- Arrays are replaced
- Null value replaces object

### Advanced fine-tuning

The configuration file also provides the possibility to change default timeouts.

Below is the list of different timeouts with their default values:

```yaml
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
```
