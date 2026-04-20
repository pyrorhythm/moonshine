package main

import (
	"log"
	"os"

	"github.com/pyrorhythm/moonshine/internal/commands"
)

func main() {
	if err := commands.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
