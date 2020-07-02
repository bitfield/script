package main

import (
	"fmt"
	"log"

	"github.com/bitfield/script"
)

// This program reads all the files supplied to it on the command line, and
// prints out the first ten lines matching the string 'error', with an 'ERROR:'
// prefix.
//
// For example:
// go run main.go errors.txt
// ERROR: Line containing the word 'error'
// ...

func main() {
	errors, err := script.Args().Concat().Match("error").First(10).Slice()
	if err != nil {
		log.Fatal(err)
	}
	for _, e := range errors {
		fmt.Printf("ERROR: %s\n", e)
	}
}
