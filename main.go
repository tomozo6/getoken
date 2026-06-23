package main

import (
	"fmt"
	"os"

	"github.com/tomozo6/getoken/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
