package pl

import (
	"fmt"
	"strings"
	"time"

	"github.com/midbel/combine"
)

const (
	DefaultMaxJobs = 256
)

type Shell struct {
	Dry     bool
	Shuffle bool
	Jobs    int
	Delay   time.Duration
}

func (s Shell) Run(args []string) error {
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

func (s Shell) runShell(ex Expander, src combine.Source) error {
	if s.Jobs <= 0 {
		s.Jobs = DefaultMaxJobs
	}
	var (
		group errgroup.Group
		sema  = make(chan struct{}, s.Jobs)
	)
	defer close(sema)
	for !src.Done() {
		sema <- struct{}{}

		time.Sleep(s.Delay)
		args, err := src.Next()
		if err != nil {
			return err
		}
		args, err = ex.Expand(args)
		if err != nil {
			return err
		}
		group.Go(func() error {
			defer func() {
				<-sema
			}()
			return nil
		})
	}
	return group.Wait()
}

func (s Shell) runDry(ex Expander, src combine.Source) error {
	for !src.Done() {
		args, err := src.Next()
		if err != nil {
			return err
		}
		args, err = ex.Expand(args)
		if err != nil {
			return err
		}
		fmt.Println(strings.Join(args, " "))
	}
	return nil
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
