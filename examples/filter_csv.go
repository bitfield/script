//go:build ignore

// This program reads a CSV file containing server names and their status (comma separated),
// filters only the lines where the status is "DOWN", and prints the name of each such server (first column).
// It uses the github.com/bitfield/script library to handle the file reading, matching, replacing, and filtering.
//
// Equivalent shell command:
// grep DOWN servers.csv | cut -d, -f1

package main

import (
	"fmt"
	"os"

	"github.com/bitfield/script"
)

func main() {
	// Check if examples/servers.csv exists, otherwise fallback to servers.csv.
	csvFile := "examples/servers.csv"
	if _, err := os.Stat(csvFile); os.IsNotExist(err) {
		csvFile = "servers.csv"
	}

	_, err := script.File(csvFile).Match("DOWN").Replace(",", " ").Column(1).Stdout()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading CSV file: %v\n", err)
		os.Exit(1)
	}
}
