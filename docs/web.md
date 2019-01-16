# Web

## DISCLAIMER: Work In Progress

## Intro

source{d} Lookout web interface provides a user friendly way to configure GitHub installations by users.


## Run the Web Interface

Since it has not been released yet, you need to build it before running it.

```shell
$ make -f Makefile.web build
$ build/bin/lookoutd web
```

Open [http://127.0.0.1:8080](http://127.0.0.1:8080) in your browser.


## Configuration

The web interface requires the usage of a Github App as authorization method, and requires GitHub App OAuth credentials.

The GitHub App authorization callback must be processed by the `/callback` frontend route, so configure it to your hostname when creating the GitHub App. The default value would be `http://127.0.0.1:8080/callback`.

Please follow the instructions on how to get the GitHub App credentials in the [main configuration guide](configuration.md#authentication-as-a-github-app), and set them in `config.yaml` as follows:

```yaml
providers:
  github:
    # Authorization with GitHub App
    app_id: 1234
    private_key: ./key.pem
    # GitHub App OAuth credentials
    client_id: 3456
    client_secret: secret-string
```

You also need to set in the `config.yaml`, the secret key which will be used to sign JSON Web Tokens.
It can be any non-empty string.

```yaml
web:
  # Secret key to sign JSON Web Tokens
  signing_key: secret123
```

There is one extra requirement. In order to identify who is an administrator, the source{d} Lookout GitHub App needs to define one extra permission:

- Organization members: Read-only


### Advanced configuration

| Env var | Option | Description | Default |
| --- | --- | --- | --- |
| `LOOKOUT_WEB_HOST` | `--host` | IP address to bind the HTTP server | `0.0.0.0` |
| `LOOKOUT_WEB_PORT` | `--port` | Port to bind the HTTP server | `8080` |
| `LOOKOUT_SERVER_URL` | `--server` | URL used to access the web server in the form 'HOSTNAME[:PORT]'. Leave it unset to allow connections from any proxy or public address | |
| `LOOKOUT_FOOTER_HTML` | `--footer` | Allows to add any custom html to the page footer. It must be a string encoded in base64. Use it, for example, to add your analytics tracking code snippet | |
