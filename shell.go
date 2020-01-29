package pl

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

const (
	DefaultMaxJobs    = 256
	DefaultMaxRetries = 1
)

type Shell struct {
	Wrap    bool
	Defer   bool
	Dry     bool
	Verbose bool
	Shuffle bool
	Jobs    int
	Retries int

	Timeout time.Duration
	Delay   time.Duration

	TempDir string
	WorkDir string

	mu sync.Mutex
}

func (s *Shell) Run(args []string) error {
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
		if src != nil {
			src.Reset()
		}
	}
	return err
}

func (s *Shell) runShell(ex Expander, src Source) error {
	ctx, cancel := context.WithCancel(context.Background())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Kill, os.Interrupt)
	go func() {
		<-sig
		cancel()
	}()

	sema := semaphore.NewWeighted(int64(s.Jobs))
	for args := range combineArgs(ex, src) {
		if err := sema.Acquire(ctx, 1); err != nil {
			return err
		}
		go func(args []string) {
			defer sema.Release(1)
			s.executeCommand(args, ctx)
		}(args)
	}
	return sema.Acquire(ctx, int64(s.Jobs))
}

func (s *Shell) executeCommand(args []string, ctx context.Context) error {
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
		var cancel context.CancelFunc
		if s.Timeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, s.Timeout)
		}
		c, err := s.prepare(args, wc, ctx)
		if err != nil {
			if cancel != nil {
				cancel()
			}
			return err
		}
		if s.Verbose && i == 0 {
			fmt.Println(strings.Join(c.Args, " "))
		}
		if err = s.runAndDump(c, wc, cancel); err == nil {
			break
		}
	}
	return err
}

func (s *Shell) dump(rc *os.File) {
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

func (s *Shell) runAndDump(c *exec.Cmd, rc *os.File, cancel context.CancelFunc) error {
	if cancel != nil {
		defer cancel()
	}
	err := c.Run()
	if rc != nil {
		s.dump(rc)
		os.Remove(rc.Name())
	}
	return err
}

func (s *Shell) prepare(args []string, w *os.File, ctx context.Context) (*exec.Cmd, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no arguments given")
	}
	c := exec.CommandContext(ctx, args[0], args[1:]...)
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

func combineArgs(ex Expander, src Source) <-chan []string {
	queue := make(chan []string)
	go func() {
		defer close(queue)
		if src == nil {
			args, err := ex.Expand(nil)
			if err == nil {
				queue <- args
			}
			return
		}
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

func splitArgs(args []string, shuffle bool) ([]Expander, Source, error) {
	var (
		src Source
		es  []Expander
		err error
	)
	if IsDelimiter(args[0]) {
		es, args, err = splitMultiple(args[1:])
	} else {
		es, args, err = splitSingle(args)
	}
	if err != nil {
		return nil, nil, err
	}
	if len(args) > 0 {
		src, err = Parse(args)
	}
	return es, src, err
}

func splitMultiple(args []string) ([]Expander, []string, error) {
	var (
		i  int
		es = make([]Expander, 0, 10)
	)
	for i = 0; i < len(args); i++ {
		if IsDelimiter(args[i]) {
			break
		}
		ws, err := Words(args[i])
		if err != nil {
			return nil, nil, err
		}
		e, err := NewExpander(ws)
		if err != nil {
			return nil, nil, err
		}
		es = append(es, e)
	}
	if i < len(args) {
		i++
	}
	return es, args[i:], nil
}

func splitSingle(args []string) ([]Expander, []string, error) {
	var i int
	for i < len(args) {
		if IsDelimiter(args[i]) {
			break
		}
		i++
	}
	if i >= len(args) {
		return nil, nil, fmt.Errorf("delimiter not found")
	}
	e, err := NewExpander(args[:i])
	if err != nil {
		return nil, nil, err
	}
	return []Expander{e}, args[i+1:], nil
}
