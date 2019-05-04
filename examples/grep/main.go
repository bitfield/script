package main

import (
	"os"

	"github.com/bitfield/script"
)

func main() {
	script.Stdin().Match(os.Args[1]).Stdout()
}
