package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/midbel/pl"
)

func main() {
	var sh pl.Shell

	flag.BoolVar(&sh.Dry, "dry", false, "dry-run")
	flag.BoolVar(&sh.Verbose, "verbose", false, "verbose")
	flag.BoolVar(&sh.Defer, "defer", false, "defer output")
	flag.BoolVar(&sh.Shuffle, "shuffle", false, "shuffle arguments")
	flag.BoolVar(&sh.Wrap, "wrap", false, "wrap linked arguments")
	flag.DurationVar(&sh.Delay, "delay", 0, "delay")
	flag.IntVar(&sh.Jobs, "jobs", 0, "jobs")
	flag.IntVar(&sh.Retries, "retries", 0, "retries")
	flag.StringVar(&sh.WorkDir, "working", "", "working directory")
	flag.StringVar(&sh.TempDir, "temp", "", "temp directory")
	flag.Parse()

	if err := sh.Run(flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "%s! abort...\n", err)
		os.Exit(1)
	}
}
