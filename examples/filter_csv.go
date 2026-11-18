//go:build ignore

// Prints the names of servers whose status is "DOWN", read from a CSV file.
//
// Run it from the repository root:
//
//	go run examples/filter_csv.go
//
// Equivalent shell command:
//
//	grep DOWN examples/servers.csv | cut -d, -f1
package main

import (
	"github.com/bitfield/script"
)

func main() {
	// Column splits on whitespace, so turn the comma into a space first,
	// then take the first field (the server name).
	_, err := script.File("examples/servers.csv").Match("DOWN").Replace(",", " ").Column(1).Stdout()
	if err != nil {
		panic(err)
	}
}
