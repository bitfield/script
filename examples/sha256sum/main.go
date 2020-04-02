package main

import (
	"fmt"
	"github.com/bitfield/script"
	"log"
	"os"
	"strings"
)

func main() {
	// Read the first 2 args, and calculate the checksum of the files.
	hashFiles, err := script.Args().First(2).SHA256Sums().String()
	if err != nil {
		log.Fatal(err)
	}

	// Compare the SHA256 checksum to check if files are identical.
	checkSums := strings.Split(hashFiles,"\n")
	if checkSums[0] == checkSums[1] {
		fmt.Println("Hashes are identical")
		os.Exit(0)
	}

	fmt.Printf("Hashes are different: %q vs %q\n", checkSums[0], checkSums[1])
	os.Exit(1)
}
