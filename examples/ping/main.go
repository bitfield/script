package main

import (
	"github.com/bitfield/script"
)

func main() {
	script.Exec("ping 127.0.0.1").Stdout()
}
