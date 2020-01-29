package pl

import (
	"bytes"
	"io"
)

const (
	space     = ' '
	squote    = '\''
	dquote    = '"'
	backslash = '\\'
	newline   = '\n'
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
		case space:
			if ws.Len() > 0 {
				xs = append(xs, ws.String())
				ws.Reset()
			}
		case squote:
		case dquote:
			var prev rune
			for {
				r, _, err := rs.ReadRune()
				if err != nil {
					return nil, err
				}
				if r == dquote && prev != backslash {
					break
				}
				ws.WriteRune(r)
				prev = r
			}
			if r, _, _ := rs.ReadRune(); r != space {
				rs.UnreadRune()
				break
			}
			if ws.Len() > 0 {
				xs = append(xs, ws.String())
				ws.Reset()
			}
		default:
			ws.WriteRune(r)
		}
	}
	if ws.Len() > 0 {
		xs = append(xs, ws.String())
	}
	return xs, nil
}
