package main

import (
	"fmt"
	"github.com/bitfield/script"
	"log"
)

func main() {
	// Read the first args, and put the 10 first lines which match the word Error.
	errors, err := script.File("./examples/slice/example.txt").Match("error").First(10).Slice()
	if err != nil {
		log.Fatal(err)
	}

	// Printing the errors
	for _, e := range errors {
		fmt.Printf("ERROR: %s\n", e)
	}
}
