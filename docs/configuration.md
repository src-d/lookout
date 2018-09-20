Configurating lookout
===

Global server configuration is stored in `config.yml` at the root directory of **lookout**. You can ask `lookoutd` to load any other config file with `--config=PATH_TO_FILE`.

The main structure of `config.yml` is:

```yml
providers:
    github:
        # configuration of github provider
repositories:
    # list of repositories to watch and user/token if needed
analyzers:
    # list of named analizers
```


# Github provider

The `providers.github` key configures how **lookout** will connect with GitHub. 

```yml
providers:
  github:
    comment_footer: "_If you have feedback about this comment, please, [tell us](%s)._"
    # app_id: 1234
    # private_key: ./key.pem
    # installation_sync_interval: 1h
```

`comment_footer` key defines a format-string that will be used for custom messages for every message posted on GitHub; see how to [add a custom message to the posted comments](#custom-footer)

<a id=basic-auth></a>
## Authentication with GitHub

It is needed to define a valid way to authenticate **lookout** with GitHub to post the analysis on any pull request of a GitHub repository.

The default method to authenticate with GitHub is using [GitHub access token](https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/), and pass the user and the token to `lookoutd`:

- user: `github-user` argument or `GITHUB_USER` enviroment variable,
- token: `github-token` argument or `GITHUB_TOKEN` enviroment variable,

<a id=github-app></a>
### Authentication as a GitHub App

Instead of using a GitHub username and token you can use **lookout** as a [GitHub App](https://developer.github.com/apps/about-apps/).

To do so, you must also unset any environment variable or option for the GitHub username and token authentication.

You need to create a GitHub App following the [documentation about creating a GitHub app](https://developer.github.com/apps/building-github-apps/creating-a-github-app/), then download a private key following the [documentation about authenticating with GitHub Apps](https://developer.github.com/apps/building-github-apps/authenticating-with-github-apps/) and set the following fields in your `config.yml` file:

```yml
providers:
  github:
    app_id: 1234
    private_key: ./key.pem
    installation_sync_interval: 1h
```

When it is used the GitHub App authentication method, the repositories to analyze are retrieved automatically from the GitHub installations, so `repositories` list from `config.yml` is ignored.

The update interval is defined by `installation_sync_interval`.


# Repositories

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

Due to the [bug#277](https://github.com/src-d/lookout/issues/277) **lookout** can parse only public repositories.


# Analyzers

Each analyzer to be requested by **lookout** should be defined under `analyzers` key.

```yml
analyzers:
  - name: Example name # required, unique name of the analyzer
    addr: ipv4://localhost:10302 # required, gRPC address
    disabled: false # optional, false by default
    feedback: http://example.com/analyzer # url to link in the comment_footer
    settings: # optional, this field is sent to analyzer "as is"
        threshold: 0.8
```

`feedback` key contains the url used in the custom footer added to any message posted on GitHub; see how to [add a custom message to the posted comments](#custom-footer)

<a id=custom-footer></a>
## Add a custom message to the posted comments

You can configure **lookout** to add a custom message to every comment that each analyzer returns. This custom message will be created following the rule:
```
Sprinf(providers.github.comment_footer, feedback)
```
If any of both is not defined, the custom message won't be added.

example:
```text
If you have feedback about this comment, please, [tell us](mailto:feedback@lookout.com)
```

## Customize an analyzer from the repository

It's possible to override Analyzers configuration for a particular repository. The new configuration to apply for certain repository will be fetched from `.lookout.yml` file at the root directory of that repository.

Example:
```yml
analyzers:
  - name: Example name
    disabled: true
    settings:
        threshold: 0.9
        mode: confident
```

The analyzer to configure must be identified with the same `name` in the `.lookout.yml` config file as in **lookout** server configuration, otherwise it will be ignored.

The repository can disable any analyzer, but it can not require new analyzers nor enable those that are disabled in the **lookout** server.

The `settings` for each analyzer in the `.lookout.yml` config file will be merged with the **lookout** configuration following these rules:

- Objects are deep merged
- Arrays are replaced
- Null value replaces object
