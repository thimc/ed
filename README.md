# ed

ed, the famous line-oriented text editor from the 70s, i.e _the
UNIX text editor_.

_NOTE: This program is missing a lot of ed's core functionality so
I wouldn't use this just yet ;-)_

Written from scratch in Go with no third party dependencies.


To get a hint of what's missing, run the following command in your shell:

    grep -rn 'TODO' *.go

# Installation

	go build -o ed
	./ed

# License

MIT
