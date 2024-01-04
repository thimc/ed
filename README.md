# ed

ed, the famous line-oriented text editor from the 70s, i.e _the
UNIX text editor_.

Written from scratch in Go with no third party dependencies.

_NOTE: This program is missing a lot of ed's core functionality so
I wouldn't use this just yet ;-)_

_NOTE 2: I know that there are a many versions of ed out there, so know
that This version of ed tries to follow the functionality of OpenBSD's ed(1)_


To get a hint of what's missing, run the following command in your shell:

    grep -rn 'TODO' *.go

# Installation

	go build -o ed
	./ed

# License

MIT
