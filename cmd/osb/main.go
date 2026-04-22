package main

import (
	"fmt"
	"os"

	"github.com/ollama-switchboard/ollama-switchboard/internal/cli"
)

func main() {
	if err := cli.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
