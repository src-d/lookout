Configurating lookout
===

Global server configuration is stored in `config.yml`, but you can specify the config file to parse with `--config=PATH_TO_FILE`. Its main structure is:

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

Under `providers.github` key you can configure how lookout will connect with GitHub. 

```yml
providers:
  github:
    comment_footer: "_If you have feedback about this comment, please, [tell us](%s)._"
    # app_id: 1234
    # private_key: ./key.pem
    # installation_sync_interval: 1h
```

For more information about the `comment_footer` key, see how to [add a custom message to the posted comments](#custom-footer)

<a id=basic-auth></a>
## Authentication with GitHub

To trigger the analysis on any pull request of a GitHub repository you will need a valid way to authenticate **lookout**.

The default method to authenticate with GitHub is using [GitHub access token](https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/), and pass the user and the token to `lookoutd`:

- user: `github-user` argument or `GITHUB_USER` enviroment variable,
- token: `github-token` argument or `GITHUB_TOKEN` enviroment variable,

<a id=github-app></a>
### Authentication as a GitHub App

Instead of using a GitHub username and token you can use **lookout** as a [GitHub App](https://developer.github.com/apps/about-apps/).

To do so, you must also unset any environment variable or option for the GitHub username and token authentication.

You need to create a GitHub App following the [documentation about creating a GitHub app](https://developer.github.com/apps/building-github-apps/creating-a-github-app/),  then download a private key following the [documentation about authenticating with GitHub Apps](https://developer.github.com/apps/building-github-apps/authenticating-with-github-apps/) and set the following fields in your `config.yml` file:

```yml
providers:
  github:
    app_id: 1234
    private_key: ./key.pem
    installation_sync_interval: 1h
```

**note**

When using this authentication method the repositories to analyze are retrieved automatically from the GitHub installations, so `repositories` list from `config.yml` is ignored.

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


# Analyzers

Each analyzer to be requested by **lookout** should be defined under `analyzers` key.

```yml
analyzers:
  - name: Example name # required, unique name of the analyzer
    addr: ipv4://localhost:10302 # required, gRPC address
    disabled: false # optional, false by default
    feedback: http://example.com/analyzer # url to link in the comment_footer. For example, to open a new GitHub issue
    settings: # optional, this field is sent to analyzer "as is"
        threshold: 0.8
```

<a id=custom-footer></a>
## Add a custom message to the posted comments

You can configure lookout to add a custom message to every comment that each analyzer returns. This custom message will be created passing the value of `feedback` key defined fot this analyzer, to the format-string defined by `providers.github.comment_footer`. If any of both is not defined, the custom message won't be added.

## Customize an analyzer from the repository

It's possible to override Analyzers configuration for a particular repository. To do that `.lookout.yml` must be present in the root of that repository.

Example:
```yml
analyzers:
  - name: Example name
    disabled: true
    settings:
        threshold: 0.9
        mode: confident
```

The `name` of the analyzer must be the same in the `.lookout.yml` config file as in **lookout** server configuration, otherwise itn will be ignored.

The repository can disable any analyzer, but it can not purpose new analyzers nor enable those that are disabled in the **lookout** server.

The `setings` for each analyzer in the `.lookout.yml` config file will be merged with the **lookout** server configuration following these rules:

- Objects are deep merged
- Arrays are replaced
- Null value replaces object


