//go:build ignore

// This program reads a log file, filters only the lines containing "ERROR", and prints the count of those lines.
// It uses the github.com/bitfield/script library to handle the file reading, matching, and counting.
//
// Equivalent shell command:
// grep ERROR app.log | wc -l

package main

import (
	"fmt"
	"os"

	"github.com/bitfield/script"
)

func main() {
	// Check if examples/app.log exists, otherwise fallback to app.log.
	logFile := "examples/app.log"
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		logFile = "app.log"
	}

	count, err := script.File(logFile).Match("ERROR").CountLines()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading log file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Number of ERROR lines: %d\n", count)
}
