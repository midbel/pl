package main

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/midbel/uuid"
)

var (
	ErrIndex = errors.New("no index")
	ErrRange = errors.New("out of range")
)

type Builder struct {
	args []Fragment

	cmd string
	env []string
}

func Build(args []string) *Builder {
	b := Builder{
		cmd: args[0],
		env: os.Environ(),
	}
	var no int
	for _, a := range args[1:] {
		f, i := parseFragment(a, no)
		b.args, no = append(b.args, f), no+i
	}
	return &b
}

func (b Builder) Dump(xs []string) (string, error) {
	as, err := b.prepareArguments(xs)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s %s", b.cmd, strings.Join(as, " ")), nil
}

func (b Builder) Build(xs []string, env, shell bool) (*exec.Cmd, error) {
	as, err := b.prepareArguments(xs)
	if err != nil {
		return nil, err
	}
	var cmd *exec.Cmd
	if shell {
		cmd = shellCommand(b.cmd, as)
	} else {
		cmd = subCommand(b.cmd, as)
	}
	if env && len(b.env) > 0 {
		cmd.Env = append(cmd.Env, b.env...)
	}
	return cmd, nil
}

func (b Builder) prepareArguments(xs []string) ([]string, error) {
	var (
		rp int
		as []string
	)
	for _, a := range b.args {
		n, s, err := a.Replace(xs)
		if err != nil {
			return nil, err
		}

		rp += n
		as = append(as, s)
	}
	if rp == 0 {
		as = append(as, xs...)
	}
	return as, nil
}

type Fragment struct {
	args    []Arg
	builder strings.Builder
}

func parseFragment(str string, no int) (Fragment, int) {
	const (
		lcurly = '{'
		rcurly = '}'
	)

	skipN := func(str string, char byte) int {
		var i int
		for i < len(str) && str[i] == char {
			i++
		}
		return i
	}
	var (
		i int
		j int
		f Fragment
	)
	for i < len(str) {
		if str[i] == lcurly {
			i++
			i += skipN(str[i:], str[i-1])
			f.appendLiteral(str[j : i-1])
			j = i
			for i < len(str) {
				if str[i] == rcurly {
					err := f.appendPlaceholder(str[j:i], no+1)
					if err == ErrIndex {
						no++
					}
					j = i + 1
					break
				}
				i++
			}
		}
		i++
	}
	if j >= 0 && j < len(str) {
		f.appendLiteral(str[j:])
	}
	return f, no
}

func (f *Fragment) String() string {
	var b strings.Builder
	for _, a := range f.args {
		if a.IsLiteral() {
			b.WriteString(a.Literal)
		} else {
			fmt.Fprintf(&b, "{%d}", a.Index)
		}
	}
	return b.String()
}

func (f *Fragment) appendLiteral(str string) {
	if len(str) > 0 {
		f.args = append(f.args, literal(str))
	}
}

func (f *Fragment) appendPlaceholder(str string, no int) error {
	var (
		arg Arg
		err error
	)
	if len(str) == 0 {
		err = ErrIndex
	} else {
		arg, err = parsePlaceholder(str)
		if err == nil && arg.Index == 0 {
			err = ErrIndex
		}
	}
	if arg.Index == 0 {
		arg.Index = int64(no)
	}
	if err == nil || err == ErrIndex {
		f.args = append(f.args, arg)
	}
	return err
}

func (f Fragment) Replace(xs []string) (int, string, error) {
	defer f.builder.Reset()

	var rp int
	for _, a := range f.args {
		if !a.IsLiteral() {
			rp++
		}
		str, err := a.Replace(xs)
		if err != nil {
			return -1, "", err
		}
		if str == "" {
			continue
		}
		f.builder.WriteString(str)
	}
	return rp, f.builder.String(), nil
}

type Arg struct {
	Literal   string
	Index     int64
	Transform func(string) string
}

func (a Arg) Replace(vs []string) (string, error) {
	if a.IsLiteral() {
		return a.Literal, nil
	}
	if a.Index < 0 {
		a.Index = int64(len(vs)) + a.Index
	} else {
		a.Index--
	}
	if a.Index < 0 || a.Index >= int64(len(vs)) {
		return "", ErrRange
	}
	v := vs[a.Index]
	if a.Transform != nil {
		v = a.Transform(v)
	}
	return v, nil
}

func (a Arg) IsLiteral() bool {
	return len(a.Literal) > 0
}

func parsePlaceholder(str string) (a Arg, err error) {
	if isPlaceholder(str) {
		str = str[1 : len(str)-1]
	}
	if len(str) == 0 {
		err = ErrIndex
	} else {
		var (
			cmd string
			idx string
		)
		ix := strings.Index(str, ":")
		if ix == 0 {
			cmd, err = str[1:], ErrIndex // only command given
		} else if ix < 0 {
			idx = str // no command given, only an index
		} else {
			idx, cmd = str[:ix], str[ix+1:] // index and command given
		}
		if len(idx) > 0 {
			a.Index, err = strconv.ParseInt(idx, 10, 64)
			if err != nil {
				return
			}
		}
		switch cmd {
		default:
		case "title":
			a.Transform = func(v string) string { return strings.Title(v) }
		case "upper":
			a.Transform = func(v string) string { return strings.ToUpper(v) }
		case "lower":
			a.Transform = func(v string) string { return strings.ToLower(v) }
		case "dir":
			a.Transform = func(v string) string { return filepath.Dir(v) }
		case "base":
			a.Transform = func(v string) string { return filepath.Base(v) }
		case "ext":
			a.Transform = func(v string) string { return filepath.Ext(v) }
		case "random":
			a.Transform = func(v string) string {
				bs := []byte(v)
				rand.Shuffle(len(bs), func(i, j int) {
					bs[i], bs[j] = bs[j], bs[i]
				})
				return string(bs)
			}
		case "uuid+url", "uuid+dns":
			a.Transform = func(v string) string {
				var ns uuid.UUID
				switch {
				case strings.HasSuffix(cmd, "rand"):
					return uuid.UUID4().String()
				case strings.HasSuffix(cmd, "url"):
					ns = uuid.URL
				case strings.HasSuffix(cmd, "dns"):
					ns = uuid.DNS
				default:
					return uuid.Nil.String()
				}
				return uuid.UUID5([]byte(v), ns).String()
			}
		}
	}
	return
}

func isPlaceholder(str string) bool {
	return str[0] == '{' && str[len(str)-1] == '}'
}

func literal(a string) Arg {
	return Arg{Literal: a}
}
