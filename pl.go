package pl

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/midbel/combine"
	"golang.org/x/sync/errgroup"
)

const (
	DefaultMaxJobs = 256
)

type Shell struct {
	Dry     bool
	Verbose bool
	Shuffle bool
	Jobs    int
	Retries int
	Delay   time.Duration
	Working string
}

func (s Shell) Run(args []string) error {
	if len(args) <= 1 {
		return fmt.Errorf("not enough arguments given")
	}
	if args[0] == ":::" {
		return s.runCommands(args[1:])
	}
	var run func(Expander, combine.Source) error
	if s.Dry {
		run = s.runDry
	} else {
		run = s.runShell
	}
	ex, src, err := splitArgs(args, s.Shuffle)
	if err == nil {
		err = run(ex, src)
	}
	return err
}

func (s Shell) runDry(ex Expander, src combine.Source) error {
	for args := range combineArgs(ex, src) {
		fmt.Println(strings.Join(args, " "))
	}
	return nil
}

func (s Shell) runCommands(args []string) error {
	if s.Jobs <= 0 {
		s.Jobs = DefaultMaxJobs
	}
	var (
		sema  = make(chan struct{}, s.Jobs)
		group errgroup.Group
	)
	defer close(sema)

	for _, a := range args {
		sema <- struct{}{}
		var (
			as  = strings.Split(a, " ")
			cmd = as[0]
		)
		if len(as) > 1 {
			as = as[1:]
		} else {
			as = as[:0]
		}
		group.Go(func() error {
			defer func() {
				<-sema
			}()
			time.Sleep(s.Delay)
			c := exec.Command(cmd, as...)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			
			if s.Verbose {
				fmt.Println(strings.Join(c.Args, " "))
			}
			return c.Run()
		})
	}
	return group.Wait()
}

func (s Shell) runShell(ex Expander, src combine.Source) error {
	if s.Jobs <= 0 {
		s.Jobs = DefaultMaxJobs
	}
	for args := range combineArgs(ex, src) {
		_ = args
	}
	return nil
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

func splitArgs(args []string, shuffle bool) (Expander, combine.Source, error) {
	var i int
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
	s, err := combine.Parse(args[i+1:])
	return e, s, err
}
