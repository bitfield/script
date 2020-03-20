package main

import (
	"fmt"
	"github.com/bitfield/script"
)

func main() {
	hashFile1, _ := script.ListFiles("./testdata/multiple_files/1.txt").SHA256Sums().String()
	hashFile2, _ := script.ListFiles("./testdata/multiple_files/2.txt").SHA256Sums().String()

	if hashFile1 == hashFile2 {
		fmt.Print("Same file.")
	} else {
		fmt.Print("Files are different.")
	}
}
