package main

import (
	"fmt"
	"github.com/bitfield/script"
	"log"
)

func main() {
	errors, err := script.Args().Concat().Match("Error").First(10).Slice()
	if err != nil {
		log.Fatal(err)
	}

	for _, e := range errors {
		fmt.Printf("ERROR: %s", e)
	}
}
