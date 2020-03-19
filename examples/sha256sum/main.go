package main

import (
	"github.com/bitfield/script"
	"os"
)

func main() {
	listPath := "."
	if len(os.Args) > 1 {
		listPath = os.Args[1]
	}
	script.ListFiles(listPath).SHA256Sum().Stdout()
}
