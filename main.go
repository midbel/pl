package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

func main() {
	var r Runner

	flag.DurationVar(&r.Delay, "delay", 0, "delay")
	flag.DurationVar(&r.Timeout, "timeout", 0, "timeout")
	flag.IntVar(&r.Repeat, "repeat", 0, "repeat")
	flag.IntVar(&r.Retries, "retries", 0, "retries")
	flag.IntVar(&r.Jobs, "jobs", 0, "jobs")
	flag.BoolVar(&r.Quiet, "quiet", false, "quiet")
	flag.BoolVar(&r.Env, "env", false, "copy env")
	flag.BoolVar(&r.Dry, "dry", false, "dry run")
	flag.BoolVar(&r.Shell, "shell", false, "shell")
	flag.BoolVar(&r.Shuffle, "shuffle", false, "shuffle")
	flag.BoolVar(&r.KeepEmpty, "keep-empty", false, "keep empty line")
	flag.Parse()

	if err := r.Run(flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "%s! abort...\n", err)
		os.Exit(1)
	}
}

var ErrIndex = errors.New("no index")

type Arg struct {
	Literal   string
	Index     int64
	Transform func(string) string
}

func (a Arg) Replace(vs []string) string {
	if a.IsLiteral() {
		return a.Literal
	}
	if a.Index < 0 {
		a.Index = int64(len(vs)) + a.Index
	}
	if i := a.Index - 1; i >= int64(len(vs)) {
		return ""
	}
	v := vs[a.Index-1]
	if a.Transform != nil {
		v = a.Transform(v)
	}
	return v
}

func (a Arg) IsLiteral() bool {
	return len(a.Literal) > 0
}

func parseArgs(args []string) (int, []Arg, error) {
	var (
		as []Arg
		no int
		ph int
	)
	for _, a := range args {
		ph++
		if a == combArg || a == linkArg {
			break
		}
		if !isPlaceholder(a) {
			as = append(as, literal(a))
		} else {
			x, err := parsePlaceholder(a)
			switch err {
			case nil:
			case ErrIndex:
				x.Index = int64(no) + 1
				no++
			default:
				return -1, nil, err
			}
			as = append(as, x)
		}
	}
	return ph, as, nil
}

func parsePlaceholder(str string) (a Arg, err error) {
	str = str[1 : len(str)-1]
	if len(str) == 0 {
		err = ErrIndex
	} else {
		var (
			cmd string
			idx string
		)
		ix := strings.Index(str, ":")
		if ix == 0 {
			cmd, err = str, ErrIndex // only command given
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

const defaultShell = "/bin/sh"

var dummy = struct{}{}

type Runner struct {
	Delay     time.Duration
	Timeout   time.Duration
	Repeat    int
	Retries   int
	Jobs      int
	Env       bool
	Quiet     bool
	Dry       bool
	Shell     bool
	Shuffle   bool
	KeepEmpty bool

	cmd  string
	args []Arg
	src  Source
}

func (r Runner) Run(args []string) error {
	if len(args) == 0 {
		return nil
	}
	r.cmd = args[0]
	if n, as, err := parseArgs(args[1:]); err != nil {
		return err
	} else {
		r.args, args = as, args[1+n:]
	}
	if len(args) > 0 {
		if r.Shuffle {
			r.src = Shuffle(args)
		} else {
			r.src = Combine(args)
		}
	} else {
		r.src = Stdin(r.KeepEmpty)
	}

	stdout, stderr := r.CombinedOutput()
	if r.Jobs <= 0 {
		r.Jobs = runtime.NumCPU()
	}
	if r.Retries <= 0 {
		r.Retries = 1
	}
	if r.Repeat <= 0 {
		r.Repeat = 1
	}
	for i := 0; i < r.Repeat; i++ {
		if err := r.run(stdout, stderr); err != nil {
			return err
		}
		if r, ok := r.src.(*Combination); ok {
			r.Reset()
		} else {
			break
		}
	}
	return nil
}

func (r Runner) run(stdout, stderr io.Writer) error {
	var (
		group errgroup.Group
		sema  = make(chan struct{}, r.Jobs)
	)
	for vs := r.src.Next(); vs != nil; vs = r.src.Next() {
		c := r.PrepareCommand(vs)
		if r.Dry {
			fmt.Printf("%s %s\n", c.Args[0], strings.Join(c.Args[1:], " "))
			continue
		}
		if r.Delay > 0 {
			time.Sleep(r.Delay)
		}
		sema <- dummy
		group.Go(func() error {
			defer func() { <-sema }()

			c.Stdout = stdout
			c.Stderr = stderr

			var err error
			for i := 0; i < r.Retries; i++ {
				if err = c.Run(); err == nil {
					break
				}
			}
			return err
		})
	}
	return group.Wait()
}

func (r Runner) CombinedOutput() (io.Writer, io.Writer) {
	var stderr, stdout io.Writer
	if !r.Quiet {
		outr, outw := io.Pipe()
		errr, errw := io.Pipe()

		go io.Copy(os.Stdout, outr)
		go io.Copy(os.Stderr, errr)

		stderr, stdout = errw, outw
	} else {
		stderr, stdout = ioutil.Discard, ioutil.Discard
	}
	return stdout, stderr
}

func (r Runner) PrepareCommand(vs []string) *exec.Cmd {
	var (
		xs []string
		ph int
	)
	for _, a := range r.args {
		if !a.IsLiteral() {
			ph++
		}
		xs = append(xs, a.Replace(vs))
		if i := len(xs) - 1; xs[i] == "" {
			xs = xs[:i]
		}
	}
	if ph == 0 {
		xs = append(xs, vs...)
	}

	var c *exec.Cmd
	if r.Shell {
		c = shellCommand(r.cmd, xs)
	} else {
		c = subCommand(r.cmd, xs)
	}
	if r.Env {
		c.Env = append(c.Env, os.Environ()...)
	}
	return c
}

func shellCommand(name string, args []string) *exec.Cmd {
	shell, ok := os.LookupEnv("SHELL")
	if !ok || shell == "" {
		shell = defaultShell
	}
	cmd := fmt.Sprintf("%s %s", name, strings.Join(args, " "))
	return exec.Command(shell, "-c", cmd)
}

func subCommand(name string, args []string) *exec.Cmd {
	return exec.Command(name, args...)
}

const (
	combArg = ":::"
	linkArg = ":::+"
)

type Source interface {
	Next() []string
}

type stdin struct {
	scan  *bufio.Scanner
	empty bool
}

func Stdin(empty bool) Source {
	s := bufio.NewScanner(os.Stdin)
	return &stdin{scan: s, empty: empty}
}

func (s *stdin) Next() []string {
	if err := s.scan.Err(); err != nil || !s.scan.Scan() {
		return nil
	}
	var vs []string

	str := s.scan.Text()
	if !s.empty && len(str) == 0 {
		return s.Next()
	}
	return append(vs, str)
}

type Combination struct {
	data  [][]string
	combi []int
	size  int
}

func Combine(as []string) Source {
	return combineAndShuffle(as, false)
}

func Shuffle(as []string) Source {
	return combineAndShuffle(as, true)
}

func combineAndShuffle(as []string, shuffle bool) *Combination {
	args := joinArgs(as)
	if shuffle {
		for i := range args {
			typ, xs := args[i][0], args[i][1:]
			rand.Shuffle(len(xs), func(i, j int) {
				xs[i], xs[j] = xs[j], xs[i]
			})
			args[i] = append([]string{typ}, xs...)
		}
	}
	c := Combination{data: args}
	c.Reset()
	return &c
}

func (c *Combination) Next() []string {
	if c.isDone() {
		return nil
	}
	c.next(c.size - 1)
	vs := make([]string, c.size)

	for i := 0; i < c.size; i++ {
		vs[i] = c.data[i][c.combi[i]]
	}
	return vs
}

func (c *Combination) next(i int) {
	if i < 0 {
		return
	}

	var reset bool
	if c.combi[i] == len(c.data[i])-1 {
		c.combi[i] = 0
	}
	if j := i - 1; (j >= 0 && !reset && c.combi[j] == 0) || c.combi[i] == 0 {
		reset = !reset
	}
	step := 1

	c.combi[i]++
	if j := i - 1; j >= 0 && isLink(c.data[i]) {
		if z := len(c.data[j]); len(c.data[i]) > z && c.combi[i] > z-1 {
			c.combi[i], step, reset = len(c.data[i])-1, 0, true
		} else {
			c.combi[j] = c.combi[i]
			step++
		}
	}
	if reset {
		c.next(i - step)
	}
}

func isCombination(data []string) bool {
	return data[0] == combArg
}

func isLink(data []string) bool {
	return data[0] == linkArg
}

func (c *Combination) isDone() bool {
	for i := c.size - 1; i >= 0; i-- {
		var ix, lim int
		if j := i - 1; j >= 0 && isLink(c.data[i]) {
			ix, lim = c.combi[i], len(c.data[i])
			if z := len(c.data[j]); z < lim {
				lim = z
			}
			i--
		} else {
			ix, lim = c.combi[i], len(c.data[i])
		}
		if ix < lim-1 {
			return false
		}
	}
	return true
}

func (c *Combination) Reset() {
	if len(c.combi) == 0 {
		c.size = len(c.data)
		c.combi = make([]int, c.size)
	}
	for i := 0; i < c.size; i++ {
		c.combi[i] = 0
	}
}

func joinArgs(args []string) [][]string {
	if len(args) == 0 {
		return nil
	}
	if !(args[0] == combArg || args[0] == linkArg) {
		args = append([]string{combArg}, args...)
	}
	var (
		as [][]string
		j  int
	)
	for i := 1; i < len(args); i++ {
		if args[i] == combArg || args[i] == linkArg {
			as = append(as, args[j:i])
			j = i
		}
	}
	return append(as, args[j:])
}
