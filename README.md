# ed

go clone of OpenBSD [ed(1)](https://man.openbsd.org/ed.1), the
famous line-oriented text editor from the 70s, i.e _the UNIX text
editor_.

Written from scratch in Go with no third party dependencies.

_NOTE: This program is missing some of ed's core functionality so I wouldn't
use this just yet ;-)_

To get a hint of what's missing, run the following command in your shell:

    grep -En '(FIXME|TODO)' *.go

# Installation

	go build -o ed
	./ed

# License

MIT
