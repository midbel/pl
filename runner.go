package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"time"

	"golang.org/x/sync/errgroup"
)

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

	builder *Builder
	source  Source
}

func (r *Runner) Run(args []string) error {
	if len(args) == 0 {
		return nil
	}
	if err := r.setupArgs(args); err != nil {
		return err
	}
	if r.Dry {
		return r.runDry()
	}

	stdout, stderr := r.CombinedOutput()
	if r.Jobs <= 0 {
		r.Jobs = runtime.NumCPU()
	}
	if r.Retries <= 0 {
		r.Retries = 1
	}
	if r.Repeat <= 0 || r.Dry {
		r.Repeat = 1
	}
	for i := 0; i < r.Repeat; i++ {
		if err := r.run(stdout, stderr); err != nil {
			return err
		}
		if c, ok := r.source.(*Combination); ok && !r.Dry {
			c.Reset()
		} else {
			break
		}
	}
	return nil
}

func (r *Runner) runDry() error {
	for vs := r.source.Next(); vs != nil; vs = r.source.Next() {
		cmd, err := r.builder.Dump(vs)
		if err != nil {
			return err
		}
		fmt.Println(cmd)
	}
	return nil
}

func (r *Runner) run(stdout, stderr io.Writer) error {
	var (
		group errgroup.Group
		sema  = make(chan struct{}, r.Jobs)
	)
	for vs := r.source.Next(); vs != nil; vs = r.source.Next() {
		c, err := r.builder.Build(vs, r.Env, r.Shell)
		if err != nil {
			return err
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

func (r *Runner) CombinedOutput() (io.Writer, io.Writer) {
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

func (r *Runner) setupArgs(args []string) error {
	parts, args := splitArgs(args)

	if b, err := Build(parts); err != nil {
		return err
	} else {
		r.builder = b
	}
	if len(args) > 0 {
		if r.Shuffle {
			r.source = Shuffle(args)
		} else {
			r.source = Combine(args)
		}
	} else {
		r.source = Stdin(r.KeepEmpty)
	}
	return nil
}

func splitArgs(args []string) ([]string, []string) {
	for i := 0; i < len(args); i++ {
		if a := args[i]; a == combArg || a == linkArg {
			return args[:i], args[i:]
		}
	}
	return args, nil
}
