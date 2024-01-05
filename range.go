package main

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"text/scanner"
	"unicode"
)

func (ed *Editor) ConsumeNumber(s *scanner.Scanner, tok *rune) (int, error) {
	var n, start, end int
	var err error

	start = s.Position.Offset
	for unicode.IsDigit(*tok) {
		*tok = s.Scan()
	}
	end = s.Position.Offset

	n, err = strconv.Atoi(string(ed.input[start:end]))
	log.Printf("ConsumeNumber(): %d\n", n)
	return n, err
}

func (ed *Editor) Range(s *scanner.Scanner, tok *rune) error {
	var mod rune
	if *tok == scanner.EOF {
		log.Printf("EOF\n")
		return nil
	}
	if *tok == '.' {
		*tok = s.Scan()
	}
	for {
		switch {
		case *tok == '.':
			*tok = s.Scan()
			return fmt.Errorf("invalid address")
		case *tok == '$':
			ed.Dot = len(ed.Lines)
			*tok = s.Scan()
		case *tok == '\'':
			*tok = s.Scan()
			var mark int = int(byte(*tok)) - 'a'
			log.Printf("Mark character: %c at index %d\n", *tok, mark)
			if *tok == scanner.EOF || int(mark) >= len(ed.Mark) {
				return fmt.Errorf("invalid mark character")
			}
			var addr int = ed.Mark[int(mark)]
			if addr == 0 {
				return fmt.Errorf("invalid address")
			}
			ed.Dot = addr
			*tok = s.Scan()
		case *tok == ' ':
			fallthrough
		case *tok == '\t':
			*tok = s.Scan()
		case *tok == '+':
			fallthrough
		case *tok == '-':
			fallthrough
		case *tok == '^':
			mod = *tok
			log.Printf("Modifier: %c\n", mod)
			*tok = s.Scan()
			if !unicode.IsDigit(*tok) {
				log.Printf("Next token is not a number\n")
				switch mod {
				case '^':
					fallthrough
				case '-':
					log.Printf("Dot-- (%d) = %d\n", ed.Dot, ed.Dot-1)
					ed.Dot--
				case '+':
					log.Printf("Dot++ (%d) = %d\n", ed.Dot, ed.Dot+1)
					ed.Dot--
				}
				return nil
			}
			fallthrough
		case unicode.IsDigit(*tok):
			if unicode.IsDigit(*tok) {
				n, err := ed.ConsumeNumber(s, tok)
				if err != nil {
					return fmt.Errorf("number out of range")
				}
				switch mod {
				case '^':
					fallthrough
				case '-':
					log.Printf("Dot (%d) - %d = %d\n", ed.Dot, n, ed.Dot-n)
					ed.Dot -= n
				case '+':
					log.Printf("Dot (%d) + %d = %d\n", ed.Dot, n, ed.Dot+n)
					ed.Dot += n
				default:
					log.Printf("Dot (%d) = %d\n", ed.Dot, n)
					ed.Dot = n
				}
				return nil
			}
		default:
			return nil
		}
	}
}

func (ed *Editor) DoRange() error {
	var s scanner.Scanner
	var tok rune

	s.Init(bytes.NewReader(ed.input))
	s.Mode = scanner.ScanRawStrings
	s.Whitespace ^= scanner.GoWhitespace
	tok = s.Scan()
	ed.s = &s
	ed.tok = &tok

	var modify bool
	switch tok {
	case '.':
		ed.Start = ed.Dot
		tok = s.Scan()
	case ',':
		fallthrough
	case '%':
		ed.Start = 1
		ed.Dot = len(ed.Lines)
		ed.End = ed.Dot
		tok = s.Scan()
	case '$':
		ed.Dot = len(ed.Lines)
		tok = s.Scan()
	case ';':
		ed.Start = ed.Dot
		ed.Dot = len(ed.Lines)
		ed.End = ed.Dot
		tok = s.Scan()
	case '?':
		fallthrough
	case '/':
		var mod rune = tok
		var search string
		tok = s.Scan()
		for tok != scanner.EOF {
			search += string(tok)
			tok = s.Scan()
		}
		log.Printf("Search %c -> \"%s\"\n", mod, search)
		if search == string(mod) || search == "" {
			log.Println("Search is the modifier")
			if ed.Search == "" {
				return fmt.Errorf("no previous pattern")
			}
			log.Printf("Previous search is \"%s\"\n", ed.Search)
			search = ed.Search
		} else if search[len(search)-1] == byte(mod) {
			search = search[:len(search)-2]
		}
		ed.Search = search
	search:
		switch mod {
		case '/':
			for i := ed.Dot; i < len(ed.Lines); i++ {
				match, err := regexp.MatchString(search, ed.Lines[i])
				if err != nil {
					return err
				}
				if match {
					log.Printf("Line %d (%s) matches the search string!\n", i, ed.Lines[i])
					ed.Dot = i + 1
					modify = true
					break search
				}
			}
			return fmt.Errorf("no match")
		case '?':
			for i := len(ed.Lines) - 1; i != ed.Dot; i-- {
				match, err := regexp.MatchString(search, ed.Lines[i])
				if err != nil {
					return err
				}
				if match {
					log.Printf("Line %d (%s) matches the search string!\n", i, ed.Lines[i])
					ed.Dot = i + 1
					modify = true
					break search
				}
			}
			return fmt.Errorf("no match")
		}
	default:
		modify = true
	}

	log.Printf("Token: %c\n", tok)
	if err := ed.Range(&s, &tok); err != nil {
		return err
	}
	if modify {
		ed.Start = ed.Dot
	}

	if tok == ',' {
		log.Println("Address 2")
		tok = s.Scan()
		log.Printf("Token: %c\n", tok)
		ed.Dot = ed.End
		ed.Range(&s, &tok)
	}
	ed.End = ed.Dot

	// if ed.Dot-1 < 0 || ed.Start-1 < 0 ||
	if ed.End < ed.Start ||
		ed.End > len(ed.Lines) || ed.Start > len(ed.Lines) {
		return fmt.Errorf("invalid address")
	}

	log.Printf("Dot=%d, Start=%d, End=%d, Token=%c\n", ed.Dot, ed.Start, ed.End, tok)

	if tok == scanner.EOF {
		log.Println("Detected no command, reverts to p on DOT")
		ed.Start = ed.Dot
		tok = 'p'
	}
	return nil
}
