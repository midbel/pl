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
	flag.BoolVar(&sh.Shuffle, "shuffle", false, "shuffle arguments")
	flag.DurationVar(&sh.Delay, "delay", 0, "delay")
	flag.IntVar(&sh.Jobs, "jobs", 0, "jobs")
	flag.Parse()

	if err := sh.Run(flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "%s! abort...\n", err)
		os.Exit(1)
	}
}
