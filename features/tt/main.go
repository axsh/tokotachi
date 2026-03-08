package main

import (
	"os"

	"github.com/axsh/tokotachi/features/devctl/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
