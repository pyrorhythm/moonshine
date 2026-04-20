package main

import (
	"log"
	"os"

	"pyrorhythm.dev/moonshine/internal/commands"
)

func main() {
	if err := commands.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
