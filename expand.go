package pl

import (
	"fmt"
	"math/rand"
	"path"
	"sort"
	"strconv"
	"strings"
)

type Expander interface {
	Expand([]string) ([]string, error)
}

func NewExpander(args []string) (Expander, error) {
	es := make([]Expander, 0, len(args))
	for _, a := range args {
		e, err := parseArgument(a)
		if err != nil {
			return nil, err
		}
		es = append(es, e)
	}
	return set{funcs: es}, nil
}

type set struct {
	funcs []Expander
}

func (s set) Expand(args []string) ([]string, error) {
	var as []string
	for _, e := range s.funcs {
		es, err := e.Expand(args)
		if err != nil {
			return nil, err
		}
		var str string
		switch len(es) {
		case 0:
		case 1:
			str = es[0]
		default:
			str = strings.Join(es, "")
		}
		as = append(as, str)
	}
	return as, nil
}

type literal string

func (i literal) Expand(_ []string) ([]string, error) {
	return []string{string(i)}, nil
}

type argument struct {
	index     int
	transform func(string) string
}

func parseArgument(str string) (Expander, error) {
	var (
		offset int
		funcs  []Expander
	)
	for {
		ix := strings.IndexByte(str[offset:], '{')
		if ix < 0 {
			break
		}
		if offset > 0 && str[offset-1] == '$' {
			offset += ix
			continue
		}
		if ix > 0 {
			funcs = append(funcs, literal(str[offset:offset+ix]))
		}
		ix++
		offset += ix
		if e, n, err := parsePlaceholder(str[offset:]); err != nil {
			return nil, err
		} else {
			offset += n
			funcs = append(funcs, e)
		}
	}
	var e Expander
	if offset == 0 {
		e = literal(str)
	} else {
		if offset < len(str) {
			funcs = append(funcs, literal(str[offset:]))
		}
		e = set{
			funcs: funcs,
		}
	}
	return e, nil
}

// syntax = {index:transform}
// syntax = {index:}
// syntax = {index}
func parsePlaceholder(str string) (Expander, int, error) {
	var (
		offset int
		arg    argument
	)
	offset = advanceUntil(str, offset, ':', '}')
	if n, err := strconv.ParseInt(str[:offset], 10, 64); err != nil {
		return nil, 0, err
	} else {
		arg.index = int(n)
	}
	switch char, pos := str[offset], offset+1; char {
	case ':':
		offset = advance(str, offset)
		if fn, err := transform(str[pos:offset]); err != nil {
			return nil, 0, err
		} else {
			arg.transform = fn
		}
	case '#': // trim prefix
		offset = advance(str, pos)
		arg.transform = trimLeft(str[pos:offset])
	case '%': // trim suffix
		offset = advance(str, pos)
		arg.transform = trimRight(str[pos:offset])
	case '}':
	default:
		return nil, 0, fmt.Errorf("invalid syntax: unexpected char %c", char)
	}
	if str[offset] != '}' {
		return nil, 0, fmt.Errorf("invalid syntax (missing closing brace)")
	}
	return arg, offset + 1, nil
}

func (a argument) Expand(vs []string) ([]string, error) {
	ix := a.index
	if ix < 0 {
		ix = len(vs) + ix
	} else {
		ix--
	}
	if ix < 0 || ix >= len(vs) {
		return nil, fmt.Errorf("invalid index %d", a.index)
	}
	str := vs[ix]
	if a.transform != nil {
		str = a.transform(str)
	}
	return []string{str}, nil
}

func advance(str string, offset int) int {
	return advanceUntil(str, offset, '}')
}

func advanceUntil(str string, offset int, set ...byte) int {
	sort.Slice(set, func(i, j int) bool {
		return set[i] < set[j]
	})
	for offset < len(str) {
		x := sort.Search(len(set), func(i int) bool {
			return set[i] >= str[offset]
		})
		if x < len(set) && set[x] == str[offset] {
			break
		}
		offset++
	}
	return offset
}

func transform(str string) (func(string) string, error) {
	var fn func(string) string
	switch strings.ToLower(str) {
	case "":
	case "lower":
		fn = strings.ToLower
	case "upper":
		fn = strings.ToUpper
	case "title":
		fn = strings.Title
	case "trim":
		fn = strings.TrimSpace
	case "random", "rand":
		fn = randomize
	case "length", "len":
		fn = length
	case "basename", "base":
		fn = path.Base
	case "dirname", "dir":
		fn = path.Dir
	case "ext":
		fn = path.Ext
	default:
		return nil, fmt.Errorf("unknown action: %s", str)
	}
	return fn, nil
}

func trimLeft(cutset string) func(string) string {
	return func(str string) string {
		return strings.TrimLeft(str, cutset)
	}
}

func trimRight(cutset string) func(string) string {
	return func(str string) string {
		return strings.TrimRight(str, cutset)
	}
}

func randomize(str string) string {
	bs := []byte(str)
	rand.Shuffle(len(bs), func(i, j int) {
		bs[i], bs[j] = bs[j], bs[i]
	})
	return string(bs)
}

func length(str string) string {
	n := len(str)
	return strconv.FormatInt(int64(n), 10)
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}
