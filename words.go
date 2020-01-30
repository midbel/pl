package pl

import (
	"bytes"
	"io"
)

const (
	space     = ' '
	tab       = '\t'
	squote    = '\''
	dquote    = '"'
	backslash = '\\'
	dollar    = '$'
	newline   = '\n'
	lparen    = '('
	rparen    = ')'
	lcurly    = '{'
	rcurly    = '}'
)

func Words(str string) ([]string, error) {
	var (
		xs []string
		ws bytes.Buffer
		rs = bytes.NewReader([]byte(str))
	)
	for {
		r, _, err := rs.ReadRune()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch r {
		case space, tab:
			xs = append(xs, ws.String())
			ws.Reset()
		case squote:
			if err := quoteStrong(rs, &ws); err != nil {
				return nil, err
			}
		case dquote:
			if err := quoteWeak(rs, &ws); err != nil {
				return nil, err
			}
		case backslash:
		case dollar:
		default:
			ws.WriteRune(r)
		}
	}
	return append(xs, ws.String()), nil
}

func quoteWeak(rs *bytes.Reader, ws *bytes.Buffer) error {
	var prev rune
	for {
		r, _, err := rs.ReadRune()
		if err != nil {
			return err
		}
		if r == dquote && prev != backslash {
			return nil
		}
		ws.WriteRune(r)
		prev = r
	}
}

func quoteStrong(rs *bytes.Reader, ws *bytes.Buffer) error {
	for {
		r, _, err := rs.ReadRune()
		if err != nil {
			return err
		}
		if r == squote {
			return nil
		}
		ws.WriteRune(r)
	}
}
