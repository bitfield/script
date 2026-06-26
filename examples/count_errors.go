//go:build ignore

// Counts the lines containing "ERROR" in a log file.
//
// Run it from the repository root:
//
//	go run examples/count_errors.go
//
// Equivalent shell command:
//
//	grep ERROR examples/app.log | wc -l
package main

import (
	"fmt"

	"github.com/bitfield/script"
)

func main() {
	count, err := script.File("examples/app.log").Match("ERROR").CountLines()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Number of ERROR lines: %d\n", count)
}
