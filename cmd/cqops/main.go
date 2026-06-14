package main

import (
	"fmt"
	"os"

	"github.com/szporwolik/cqops/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "\nCQOps cannot start: %v\n\n", err)
		os.Exit(1)
	}
}
