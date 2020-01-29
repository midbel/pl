package pl

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/midbel/combine"
	"golang.org/x/sync/semaphore"
)

const (
	DefaultMaxJobs    = 256
	DefaultMaxRetries = 256
)

const (
	prefixOut = '<'
	prefixErr = '>'
)

type Shell struct {
	Defer   bool
	Dry     bool
	Verbose bool
	Shuffle bool
	Jobs    int
	Retries int
	Delay   time.Duration

	TempDir string
	WorkDir string

	mu sync.Mutex
}

func (s Shell) Run(args []string) error {
	if len(args) <= 1 {
		return fmt.Errorf("not enough arguments given")
	}
	if s.Jobs <= 0 {
		s.Jobs = DefaultMaxJobs
	}
	if s.Retries <= 0 {
		s.Retries = DefaultMaxRetries
	}
	exs, src, err := splitArgs(args, s.Shuffle)
	if err != nil {
		return err
	}
	for _, e := range exs {
		if err = s.runShell(e, src); err != nil {
			break
		}
		src.Reset()
	}
	return err
}

func (s Shell) runShell(ex Expander, src combine.Source) error {
	var (
		sema = semaphore.NewWeighted(int64(s.Jobs))
		ctx  = context.TODO()
	)
	for args := range combineArgs(ex, src) {
		if err := sema.Acquire(ctx, 1); err != nil {
			return err
		}
		go func(args []string) {
			defer sema.Release(1)
			s.executeCommand(args)
		}(args)
	}
	return sema.Acquire(ctx, int64(s.Jobs))
}

func (s Shell) runCommands(args []string) error {
	var (
		sema = semaphore.NewWeighted(int64(s.Jobs))
		ctx  = context.TODO()
	)
	for _, a := range args {
		if err := sema.Acquire(ctx, 1); err != nil {
			return err
		}
		go func(args []string) {
			defer sema.Release(1)
			s.executeCommand(args)
		}(strings.Split(a, " "))
	}
	return sema.Acquire(ctx, int64(s.Jobs))
}

func (s Shell) executeCommand(args []string) error {
	if s.Dry {
		fmt.Println(strings.Join(args, " "))
		return nil
	}
	time.Sleep(s.Delay)

	var err error
	for i := 0; i < s.Retries; i++ {
		var wc *os.File
		if s.Defer {
			w, err := ioutil.TempFile(s.TempDir, "pl_*.tmp")
			if err != nil {
				return err
			}
			wc = w
		}
		c, err := s.prepare(args, wc)
		if err != nil {
			return err
		}
		if s.Verbose && i == 0 {
			fmt.Println(strings.Join(c.Args, " "))
		}
		if err = s.runAndDump(c, wc); err == nil {
			break
		}
	}
	return err
}

func (s Shell) dump(rc *os.File) {
	defer rc.Close()
	if _, err := rc.Seek(0, io.SeekStart); err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	scan := bufio.NewScanner(rc)
	for scan.Scan() {
		var (
			xs = scan.Bytes()
			wr io.Writer
		)
		if len(xs) <= 1 {
			continue
		}

		if xs[0] == prefixOut {
			wr = os.Stdout
		} else {
			wr = os.Stderr
		}
		wr.Write(xs[1:])
		wr.Write([]byte("\r\n"))
	}
}

func (s Shell) runAndDump(c *exec.Cmd, rc *os.File) error {
	err := c.Run()
	if rc != nil {
		s.dump(rc)
		os.Remove(rc.Name())
	}
	return err
}

func (s Shell) prepare(args []string, w *os.File) (*exec.Cmd, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no arguments given")
	}
	c := exec.Command(args[0], args[1:]...)
	c.Dir = s.WorkDir
	if w == nil {
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
	} else {
		c.Stdout = stdout(w)
		c.Stderr = stderr(w)
	}
	return c, nil
}

type writer struct {
	tag    string
	prefix byte

	mu    sync.Mutex
	inner io.Writer
}

func stdout(w io.Writer) io.Writer {
	return writer{
		prefix: prefixOut,
		inner:  w,
	}
}

func stderr(w io.Writer) io.Writer {
	return writer{
		prefix: prefixErr,
		inner:  w,
	}
}

func (w writer) Write(xs []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.tag != "" {
		_, err := io.WriteString(w.inner, w.tag)
		if err != nil {
			return 0, err
		}
	}

	bs := make([]byte, 0, len(xs)+1)
	bs = append(bs, w.prefix)
	bs = append(bs, xs...)

	_, err := w.inner.Write(bs)
	return len(xs), err
}

func combineArgs(ex Expander, src combine.Source) <-chan []string {
	queue := make(chan []string)
	go func() {
		defer close(queue)
		for !src.Done() {
			args, err := src.Next()
			if err != nil {
				return
			}
			args, err = ex.Expand(args)
			if err != nil {
				continue
			}
			queue <- args
		}
	}()
	return queue
}

func splitArgs(args []string, shuffle bool) ([]Expander, combine.Source, error) {
	var (
		i  int
		es []Expander
	)
	if combine.IsDelimiter(args[0]) {
		es = make([]Expander, 0, 10)
		for i = 1; i < len(args); i++ {
			if combine.IsDelimiter(args[i]) {
				break
			}
			ws, err := Words(args[i])
			if err != nil {
				return nil, nil, err
			}
			e, err := Parse(ws)
			if err != nil {
				return nil, nil, err
			}
			es = append(es, e)
		}
	} else {
		for i < len(args) {
			if combine.IsDelimiter(args[i]) {
				break
			}
			i++
		}
		if i >= len(args) {
			return nil, nil, fmt.Errorf("delimiter not found")
		}
		e, err := Parse(args[:i])
		if err != nil {
			return nil, nil, err
		}
		es = []Expander{e}
	}
	s, err := combine.Parse(args[i+1:])
	return es, s, err
}
