# ed

Go clone of OpenBSD [ed(1)](https://man.openbsd.org/ed.1), the
famous line-oriented text editor that is originally from the 1970s.
Simply put, _the UNIX text editor_.

Written from scratch in Go with no third party dependencies.

_NOTE: This program is passing over 200 unit tests but I still
wouldn't trust it for production use. Be sure to make copies of
your files if you're brave enough to try it. Fun fact: Roughly 50
% of the code is unit tests._

## Installation

	go build -o ed
	./ed
