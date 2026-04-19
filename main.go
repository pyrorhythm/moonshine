package main

import (
	"context"
	"fmt"
	"os"

	"github.com/pyrorhythm/moonshine/cmd"
)

func main() {
	if err := cmd.Execute(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}