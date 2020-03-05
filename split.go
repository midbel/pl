package pl

import (
	"bytes"
	"errors"
	"io"
	"os"
)

const (
	space     = ' '
	tab       = '\t'
	squote    = '\''
	dquote    = '"'
	backslash = '\\'
	dollar    = '$'
)

func Split(str string) ([]string, error) {
	var (
		xs []string
		ws bytes.Buffer
		rs = bytes.NewReader([]byte(str))
	)
	for {
		r, _, err := rs.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		switch r {
		case space, tab:
			xs = append(xs, ws.String())
			ws.Reset()
			skipBlanks(rs)
		case squote:
			if err := scanStrong(rs, &ws); err != nil {
				return nil, err
			}
		case dquote:
			if err := scanWeak(rs, &ws); err != nil {
				return nil, err
			}
		case backslash:
		case dollar:
			str := scanVariable(rs)
			ws.WriteString(str)
		default:
			ws.WriteRune(r)
		}
	}
	return append(xs, ws.String()), nil
}

func scanVariable(rs *bytes.Reader) string {
	var buf bytes.Buffer
	buf.WriteRune(dollar)
	for {
		k, _, err := rs.ReadRune()
		if err != nil || k == space || k == tab || k == dquote {
			rs.UnreadRune()
			break
		}
		buf.WriteRune(k)
	}
	return os.ExpandEnv(buf.String())
}

func scanWeak(rs *bytes.Reader, ws *bytes.Buffer) error {
	var prev rune
	for {
		r, _, err := rs.ReadRune()
		if err != nil {
			return err
		}
		if r == dquote && prev != backslash {
			return nil
		}
		if r == dollar {
			str := scanVariable(rs)
			ws.WriteString(str)
			continue
		}
		ws.WriteRune(r)
		prev = r
	}
}

func scanStrong(rs *bytes.Reader, ws *bytes.Buffer) error {
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

func skipBlanks(rs *bytes.Reader) {
	for {
		k, _, err := rs.ReadRune()
		if err != nil {
			break
		}
		if k != space && k != tab {
			rs.UnreadRune()
			break
		}
	}
}
