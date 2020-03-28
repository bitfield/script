package main

import (
	"github.com/bitfield/script"
)

func main() {
	script.Exec("bash -c 'echo hello world'").Stdout()
}
