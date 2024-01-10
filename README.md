# ed

Go clone of OpenBSD [ed(1)](https://man.openbsd.org/ed.1), the
famous line-oriented text editor from the 70s. Simply put, _the
UNIX text editor_.

Written from scratch in Go with no third party dependencies.



_NOTE: This program is passing over 200 unit tests but I still
wouldn't trust it for production use. Be sure to make copies of
your files if you're brave enough to try it._

To get a sense of what's missing or misbehaving, run the following
command in your shell:

    grep -En '(FIXME|TODO)' *.go

To run the tests for yourself:

    go test -v

Fun fact: Roughly 50 % of the code is unit tests.

# Installation

	go build -o ed
	./ed

# License

MIT
