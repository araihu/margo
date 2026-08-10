package main

import (
	"context"
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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
