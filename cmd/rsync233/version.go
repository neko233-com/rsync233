package main

import (
	"fmt"
	"io"
	"runtime"
)

var version = "dev"

func runVersion(stdout io.Writer) {
	fmt.Fprintf(stdout, "rsync233 %s\n", version)
	fmt.Fprintf(stdout, "Go version: %s\n", runtime.Version())
	fmt.Fprintf(stdout, "Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}
