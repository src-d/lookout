# Examples

_Please refer to [**lookout-tool** docs](lookout-tool.md) to see how to locally test an analyzer without accessing GitHub._

# language-analyzer

`language-analyzer` is an example of an analyzer that detects the language and number of functions for every modified file. It has been implemented with Golang and with Python to serve as a canonical example of how to program your own analyzer.

You can find its code in [lookout-sdk/examples](https://github.com/src-d/lookout-sdk/blob/master/examples). You can also run it from sources doing as it follows:

## Golang

You can execute [language-analyzer.go](https://github.com/src-d/lookout-sdk/blob/master/examples/language-analyzer.go) running from the **lookout-sdk** directory:

```shell
$ go get -u examples
$ go run examples/language-analyzer.go
```

## Python

You can execute [language-analyzer.py](https://github.com/src-d/lookout-sdk/blob/master/examples/language-analyzer.py) running from the **lookout-sdk** directory:

```shell
$ pip install lookout-sdk
$ python3 examples/language-analyzer.py
```

# dummy Analyzer

`dummy` is a simple analyzer that:
- warns if the modified line is longuer than a limit, and
- informs about the number of lines in which the changed file was increased.

It is used for internal testing purposes, but you can also use it when trying **source{d} Lookout**.

You can download it from [**source{d} Lookout** releases page](https://github.com/src-d/lookout/releases) and then run it:

```shell
$ dummy serve
```
