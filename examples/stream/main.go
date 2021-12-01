package main

import (
	"fmt"

	"github.com/bitfield/script"
)

// This program runs tests on the script library 50 times and prints out the progress nicely.

func main() {
	round, step := 10, 5
	progressInfo := make([]string, round)
	for i := 0; i < round; i++ {
		progressInfo[i] = fmt.Sprintf("------ Done %v / %v ------", (i+1)*step, round*step)
	}
	cmd := fmt.Sprintf("bash -c 'go test -count %v github.com/bitfield/script; echo {{.}}'", step)
	// with Stream(), the program can print to stdout in real time
	script.Slice(progressInfo).Stream().ExecForEach(cmd).Stdout()
}
