package main

import (
	"fmt"
	"os"

	"agate/client/internal/ui"
)

func main() {
	if err := ui.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "agate client exited with error: %v\n", err)
		os.Exit(1)
	}
}
