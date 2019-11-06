package main

import (
	"log"
	"os"

	"github.com/bitfield/script"
)

func main() {
	listPath := "."
	if len(os.Args) > 1 {
		listPath = os.Args[1]
	}
	_, err := script.FindFiles(listPath).Stdout()
	if err != nil {
		log.Fatal(err)
	}
}
