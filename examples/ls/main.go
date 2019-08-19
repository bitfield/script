package main

import (
	"os"
	"regexp"

	"github.com/bitfield/script"
)

// Like Unix `ls`, hide files starting with '.'
var hideDotFiles = regexp.MustCompile(`[^/]*/\\.[^/]*$`)

func main() {
	listPath := "."
	if len(os.Args) > 1 {
		listPath = os.Args[1]
	}
	script.ListFiles(listPath).RejectRegexp(hideDotFiles).Stdout()
}
