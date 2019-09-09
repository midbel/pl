package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/midbel/pl"
)

func main() {
	var r pl.Runner

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
