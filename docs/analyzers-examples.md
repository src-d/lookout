# Examples

_Please refer to [**lookout-sdk** docs](lookout-sdk.md) to see how to locally test an analyzer without accessing GitHub._

**lookout-sdk** repository contains a [quickstart example](https://github.com/src-d/lookout-sdk/blob/master/examples) &mdash;implemented in Go and in Python&mdash; of an Analyzer that detects the language and number of functions for every file.

## Golang

You can run [language-analyzer.go](https://github.com/src-d/lookout-sdk/blob/master/examples/language-analyzer.go) running from **lookout-sdk** directory:

```shell
$ go get -u examples
$ go run examples/language-analyzer.go
```

## Python

You can run [language-analyzer.py](https://github.com/src-d/lookout-sdk/blob/master/examples/language-analyzer.py) running from **lookout-sdk** directory:

```shell
$ pip install lookout-sdk
$ python3 examples/language-analyzer.py
```
