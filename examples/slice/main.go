package main

import (
	"fmt"
	"github.com/bitfield/script"
	"log"
)

func main() {
	// Read args, and return it in a Slice.
	params, err := script.Args().Slice()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Input params are %q", params)
}
