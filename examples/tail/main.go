package main

import (
	"log"
	"os"
	"strconv"

	"github.com/bitfield/script"
)

func main() {
	lines, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	script.Stdin().Last(lines).Stdout()
}
