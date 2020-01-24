package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/midbel/pl"
)

func main() {
	var sh pl.Shell
	
	flag.Parse()

	if err := sh.Run(flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "%s! abort...\n", err)
		os.Exit(1)
	}
}
