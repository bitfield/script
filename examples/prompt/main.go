package main

import (
	"strings"

	"github.com/bitfield/script"
)

func Install(path string) {
	script.Echo("📂 Installing at " + path + "\n").Stdout()
	script.Echo("🟢 Successfully installed\n").Stdout()
}

func main() {
	script.Echo("Installation of Fake Product\n").Stdout()
	curDir, err := script.Exec("pwd").String()
	if err != nil {
		panic(err)
	}

	path, _ := script.Echo("Choose install location").ReadInput(strings.TrimSpace(curDir)).String()
	Install(path)
}
