package main

import (
	"log"
	"regexp"
	"text/scanner"
	"unicode"
)

func (ed *Editor) Range() (int, error) {
	ed.addr = ed.Dot
	var mod rune
	var first bool

	for first = true; ; first = false {
		for {
			switch ed.token() {
			case ' ':
				fallthrough
			case '\t':
				fallthrough
			case '\n':
				fallthrough
			case '\r':
				ed.nextToken()
				continue
			}
			break
		}
		log.Printf("Token:'%c' (first=%t)\n", ed.token(), first)

		switch {
		case ed.token() == '.':
			fallthrough
		case ed.token() == '$':
			if !first {
				return 0, ErrInvalidAddress
			}
			if ed.token() == '.' {
				ed.addr = ed.Dot
			} else {
				ed.addr = len(ed.Lines)
			}
			ed.nextToken()

		case ed.token() == '?':
			fallthrough
		case ed.token() == '/':
			var mod rune = ed.token()
			ed.nextToken()
			var search string = ed.scanString()
			if search == string(mod) || search == "" {
				log.Println("Search with previous pattern")
				if ed.Search == "" {
					return 0, ErrNoPrevPattern
				}
				log.Printf("Previous search is \"%s\"\n", ed.Search)
				search = ed.Search
			} else if search[len(search)-1] == byte(mod) {
				search = search[:len(search)-1]
			}
			log.Printf("Search %c -> \"%s\"\n", mod, search)
			ed.Search = search
			var s int = ed.End - 1
			var e = len(ed.Lines)
			if mod == '?' {
				e = 0
			}
			ed.dump()
			log.Printf("Search start: %d, end: %d\n", s, e)
			for i := s; i != e; {
				if i < 0 || i > len(ed.Lines) {
					return 0, ErrNoMatch
				}
				match, err := regexp.MatchString(search, ed.Lines[i])
				if err != nil {
					return 0, err
				}
				if match {
					log.Printf("Line %d (%s) matches the search string!\n", i, ed.Lines[i])
					ed.addr = i + 1
					return ed.addr, nil
				}
				if mod == '/' {
					i++
				} else {
					i--
				}
			}
			return 0, ErrNoMatch

		case ed.token() == '\'':
			ed.nextToken()
			var buf string = ed.scanString()
			switch len(buf) {
			case 1:
				break
			case 0:
				fallthrough
			default:
				return 0, ErrInvalidCmdSuffix
			}
			var r rune = rune(buf[0])
			if !unicode.IsLower(r) {
				return 0, ErrInvalidMark
			}
			log.Printf("Mark %c\n", r)
			var mark int = int(byte(buf[0])) - 'a'
			if mark >= len(ed.Mark) {
				return 0, ErrInvalidMark
			}
			var maddr int = ed.Mark[mark]
			log.Printf("Mark %c address %d\n", rune('a'+mark), maddr)
			if maddr < 1 || maddr > len(ed.Lines) {
				return 0, ErrInvalidAddress
			}
			ed.End = maddr
			ed.Start = maddr
			ed.addr = maddr

		case ed.token() == '+':
			fallthrough
		case ed.token() == '-':
			fallthrough
		case ed.token() == '^':
			mod = ed.token()
			log.Printf("Modifier: %c\n", mod)
			ed.nextToken()
			if !unicode.IsDigit(ed.token()) {
				log.Printf("Next token is not a number\n")
				switch mod {
				case '^', '-':
					log.Printf("Dot-- (%d) = %d\n", ed.Dot, ed.Dot-1)
					ed.addr--
				case '+':
					log.Printf("Dot++ (%d) = %d\n", ed.Dot, ed.Dot+1)
					ed.addr++
				}
			}
			fallthrough
		case unicode.IsDigit(ed.token()):
			if !first {
				return 0, ErrInvalidAddress
			}
			if unicode.IsDigit(ed.token()) {
				log.Printf("First token is number: %c\n", ed.token())
				n, err := ed.scanNumber()
				if err != nil {
					return 0, ErrInvalidNumber
				}
				switch mod {
				case '^', '-':
					log.Printf("Addr (%d) - %d = %d\n", ed.addr, n, ed.addr-n)
					ed.addr -= n
				case '+':
					log.Printf("Addr (%d) + %d = %d\n", ed.addr, n, ed.addr+n)
					ed.addr += n
				default:
					log.Printf("Addr (%d) = %d\n", ed.addr, n)
					ed.addr = n
				}
			}

		case ed.token() == ';':
			fallthrough
		case ed.token() == '%':
			fallthrough
		case ed.token() == ',':
			log.Printf("token='%c' peek='%c' first=%t\n", ed.token(), ed.s.Peek(), first)
			var r rune = ed.token()
			if first {
				ed.nextToken()
				var err error
				n, err := ed.Range()
				if err != nil {
					return 0, err
				}
				ed.addr = n
				if n == ed.Dot && ed.Start == ed.End {
					ed.Start = 1
					if r == ';' {
						ed.Start = ed.End
					}
					ed.Dot = len(ed.Lines)
					ed.addr = ed.Dot
					ed.End = ed.Dot
					ed.addrcount = 2
					return -1, nil
				}
				continue
			}
			fallthrough

		default:
			if ed.addr < 0 || ed.addr > len(ed.Lines) {
				return -1, ErrInvalidAddress
			}
			return ed.addr, nil
		}
	}
}

func (ed *Editor) DoRange() error {
	var n int
	var err error
	ed.addrcount = 0

	ed.Start = ed.Dot
	ed.End = ed.Dot

	if ed.token() == scanner.EOF {
		goto end
	}
	for {
		n, err = ed.Range()
		if n < 0 {
			break
		}
		if err != nil {
			return err
		}
		ed.addr = n
		ed.addrcount++
		log.Printf("Start (%d) = End (%d)\n", ed.Start, ed.End)
		ed.Start = ed.End
		log.Printf("End (%d) = Addr (%d)\n", ed.End, ed.addr)
		ed.End = ed.addr

		if ed.token() != ',' && ed.token() != ';' {
			break
		} else if ed.s.Peek() == ';' {
			ed.Dot = ed.addr
		}
		if ed.token() == scanner.EOF {
			break
		}
	}

end:

	if ed.addrcount == 1 || ed.End != ed.addr {
		log.Printf("Start (%d) = End (%d)\n", ed.Start, ed.End)
		ed.Start = ed.End
	}

	log.Printf("Dot (%d) = End (%d)\n", ed.Dot, ed.End)
	ed.Dot = ed.End

	if ed.token() == scanner.EOF && ed.s.Pos().Offset == 0 {
		ed.Dot++
		ed.Start = ed.Dot
		ed.End = ed.Dot
	}

	if err := ed.checkRange(); err != nil {
		return err
	}

	return nil
}
