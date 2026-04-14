package main

import (
	"os"
)

// Build-time variables injected via ldflags.
var (
	version   = "dev"
	commit    = "none"
	date      = "unknown"
	goVersion = "unknown"
)

func main() {
	if err := executeCLI(os.Args[1:], os.Stdout, os.Stderr, os.Getenv); err != nil {
		_, _ = os.Stderr.WriteString("error: " + err.Error() + "\n")
		os.Exit(1)
	}
}
