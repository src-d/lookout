# Configuring source{d} Lookout

The behavior of **source{d} Lookout** is defined with two different configuration files:

- [`config.yml`](#config-yml), to define the global configuration of the server.
- [`.lookout.yml`](#lookout-yml), to override the analyzer default behavior in each repository.


# config.yml

**source{d} Lookout** is configured with the `config.yml` file, you can use the template [`config.yml.tpl`](/config.yml.tpl) to create your own. Use the `lookoutd` option `--config` to set the path to it, or use the default location at `./config.yml`. The config file is read on server startup, so `lookoutd` needs to be restarted in order to load a new configuration.

The most important things you need to configure for a local installation, are:

1. [Repositories](#repositories): define the repositories to be watched.
1. [Analyzers](#analyzers): add the gRPC addresses of the analyzers to be used by **source{d} Lookout**.

The main structure of `config.yml` is:

```yaml
providers:
    github:
        # configuration of GitHub provider
repositories:
    # list of repositories to watch and user/token if needed
analyzers:
    # list of named analyzers
timeout:
    # configuration for the existing timeouts.
```

For more fine grained configuration, you should pay attention to the following documentation.


## Github Provider

The `providers.github` key configures how **source{d} Lookout** will connect with GitHub.

```yaml
providers:
  github:
    comment_footer: "_Comment made by '{{.Name}}'{{with .Feedback}}, [tell us]({{.}}){{end}}._"
    # app_id: 1234
    # private_key: ./key.pem
    # installation_sync_interval: 1h
    # watch_min_interval: 2s
```

`comment_footer` key defines the [go template](https://golang.org/pkg/text/template) that will be used for custom messages for every message posted on GitHub; see how to [add a custom message to the posted comments](#add-a-custom-message-to-the-posted-comments)

### Authentication with GitHub

**source{d} Lookout** needs to authenticate with GitHub. There are two ways to authenticate with GitHub:

- Using GitHub personal access tokens.
- Authenticating as a GitHub App.

Both are explained below.

#### Authentication with GitHub personal access tokens

The easiest method for testing purposes is using a [GitHub personal access token](https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/). The token should have the `repo` scope enabled.

The credentials can be passed to `lookoutd`:
- globally for all watched repositories, using command line arguments or environment variables when running `lookoutd`:
  - user: `--github-user` argument or `GITHUB_USER` environment variable.
  - token: `--github-token` argument or `GITHUB_TOKEN` environment variable.
- per watched repository, following the instructions given by the [Repositories section](#repositories) of this docs.

#### Authentication as a GitHub App

For a production environment, you can use **source{d} Lookout** as a [GitHub App](https://developer.github.com/apps/about-apps/).

To do so, you must also unset any environment variable or option for the GitHub username and token authentication.

You need to create a GitHub App following the [documentation about creating a GitHub App](https://developer.github.com/apps/building-github-apps/creating-a-github-app/). The permissions that must be set are:

- Repository metadata: Read-only
- Pull requests: Read & write
- Repository contents: Read-only
- Commit statuses: Read & write

Download a private key following the [documentation about authenticating with GitHub Apps](https://developer.github.com/apps/building-github-apps/authenticating-with-github-apps/) and set the following fields in your `config.yml` file:

```yaml
providers:
  github:
    app_id: 1234
    private_key: ./key.pem
    installation_sync_interval: 1h
    watch_min_interval: 10s
```

Then you need to install the GitHub App in all your repositories or in a list of them.

When the GitHub App authentication method is used, the repositories to analyze are retrieved automatically from the GitHub installations, so `repositories` list from `config.yml` is ignored.

The update interval to discover new installations and repositories is defined by `installation_sync_interval`.

The minimum watch interval to discover new pull requests and push events is defined by `watch_min_interval`.

#### Web Interface

The **source{d} Lookout** Web Interface to manage the installations of your GitHub App is currently under development, but you can find more details about it and its configuration at [Web Interface docs](web.md)


## Repositories

The list of repositories to be watched by **source{d} Lookout** is defined by:
- the `repositories` field at `config.yml`, or
- the repositories where the GitHub App is installed if you [authenticated as a GitHub App](#authentication-as-a-github-app). In that case, the `repositories` field in `config.yml` will be ignored.

The user and token to be used for the Github authentication can be defined per repository; if you do so, it will override the globally passed user and token.

```yaml
repositories:
  - url: github.com/<user>/<repo1>
  - url: github.com/<user>/<repo2>
    client:
      # user: github-user
      # token: github-user-token
      # minInterval: 1m
```


## Analyzers

Each analyzer to be requested by **source{d} Lookout** should be defined under `analyzers` key.

```yaml
analyzers:
  - name: Example name # required, unique name of the analyzer
    addr: ipv4://localhost:9930 # required, gRPC address
    disabled: false # optional, false by default
    feedback: http://example.com/analyzer # url to link in the comment_footer
    settings: # optional, this field is sent to analyzer "as is"
        threshold: 0.8
```

`feedback` key contains the URL used in the custom footer added to any message posted on GitHub; see how to [add a custom message to the posted comments](#add-a-custom-message-to-the-posted-comments)

### Add a Custom Message to the Posted Comments

You can configure **source{d} Lookout** to add a custom message to every comment that each analyzer returns. This custom message will be created from the template defined by `providers.github.comment_footer`, using the configuration set for each analyzer.

If the template (`providers.github.comment_footer`) is empty, or the analyzer configuration does not define any of the values that the template requires, the custom message won't be added.

For example, for this configuration, each analyzer needs to define `name` and `settings.email`:

```yaml
providers:
  github:
    comment_footer: "Comment made by analyzer {{.Name}}, [email me]({{.Settings.email}})."

analyzers:
  - name: Fancy Analyzer
    addr: ipv4://localhost:9930
    settings:
      email: admin@example.org
  - name: Awesome Analyzer
    addr: ipv4://localhost:9931
```

Comments from `Fancy Analyzer` will have this footer appended:
>_Comment made by analyzer Fancy Analyzer, [email me](admin@example.org)._

but comments from `Awesome Analyzer` wont have a footer message because in its configuration it's missing the `settings.email` value.


## Timeouts

The timeouts used by `lookoutd` for some operations can be modified or disabled from the `config.yml` file.

If any timeout is set to `0`, there will be no timeout for that process.

Below is the list of different timeouts in **source{d} Lookout**, with their default values:

```yaml
# These are the default timeout values. A value of 0 means no timeout
timeout:
  # Timeout for an analyzer to reply a NotifyReviewEvent
  analyzer_review: 10m
  # Timeout for an analyzer to reply a NotifyPushEvent
  analyzer_push: 60m
  # Timeout for HTTP requests to the GitHub API
  github_request: 1m
  # Timeout for Git fetch actions
  git_fetch: 20m
  # Timeout for Bblfsh to reply to a Parse request
  bblfsh_parse: 2m
```


# .lookout.yml

It's possible to customize the Analyzers configuration for each repository. To do that you only need to place a `.lookout.yml` file at the root directory of that repository.

Example:
```yaml
analyzers:
  - name: Example name
    disabled: true
    settings:
        threshold: 0.9
        mode: confident
```

The analyzer to configure must be identified with the same `name` in the `.lookout.yml` config file as in **source{d} Lookout** server configuration, otherwise, it will be ignored.

The repository can disable any analyzer, but it cannot define new analyzers nor enable those that are disabled in the **source{d} Lookout** server.

The `settings` for each analyzer in the `.lookout.yml` config file will be merged with the **source{d} Lookout** configuration following these rules:

- Objects are deep merged
- Arrays are replaced
- Null value replaces object
