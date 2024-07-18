# ed

Go clone of [ed(1)](https://man.openbsd.org/ed.1), the famous
line-oriented text editor that is originally from the 1970s. Simply
put, _the UNIX text editor_.

## Differences
This version of ed aims to be a bug for bug implementation of the
original implementation. The only thing that differs is that this
version uses RE2 instead of BRE (basic regular expresions). The reason
for this is that the Go programming languages standard library uses
that in the [regexp](https://pkg.go.dev/regexp) package.

Written from scratch in Go with no third party dependencies.

## TODO

	godoc -notes 'TODO'

## Installation

	go build
	./ed file
