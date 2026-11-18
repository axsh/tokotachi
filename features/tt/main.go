package main

import (
	"errors"
	"os"

	"github.com/axsh/tokotachi/features/tt/cmd"
	"github.com/axsh/tokotachi/features/tt/internal/kdb/jsonout"
)

func main() {
	if err := cmd.Execute(); err != nil {
		var ee *jsonout.ExitError
		if errors.As(err, &ee) && ee.Code > 0 {
			os.Exit(ee.Code)
		}
		os.Exit(1)
	}
}
