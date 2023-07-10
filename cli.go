package main

import "os"

// CLI responsible for processing command line arguments

func validateArgs() {
	// cli arg len must be greater than 1
	if len(os.Args) < 2 {
		os.Exit(1)
	}
}


type CLI struct {
}
