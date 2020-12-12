package main

import (
	"strings"

	"github.com/bitfield/script"
)

func main() {
	script.Echo("Installation of Fake Product\n").Stdout()
	curDir, err := script.Exec("pwd").String()
	if err != nil {
		panic(err)
	}

	script.Echo("Choose install location").ReadInput(strings.TrimSpace(curDir)).ExecForEach("echo Installing at {{.}}").Stdout()
}
