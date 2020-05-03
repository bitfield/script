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

	script.Prompt("Choose install location: ", strings.TrimSpace(curDir)).
		ExecForEach("touch {{.}}/fake_install.sh").Stdout()

	script.Echo("Installation of Fake Product complete").Stdout()
}
