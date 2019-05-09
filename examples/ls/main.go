package main

import (
	"github.com/bitfield/script"
)

func main() {
	script.Args().ConcatFiles().Stdout()
}
