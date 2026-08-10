package main

import (
	"context"
	"errors"
	"fmt"
	"os"
)

func main() {
	err := Execute(context.Background(), Dependencies{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Build:  ReadBuildInfo(nil),
	})
	if err != nil {
		var reported reportedError
		if !errors.As(err, &reported) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
