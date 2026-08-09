package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/araihu/margo/examples/blog/site"
)

func main() {
	output := flag.String("out", "examples/blog/generated", "directory for generated blog pages")
	flag.Parse()
	if err := site.Generate(*output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
