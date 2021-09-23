package main

import (
	"bufio"
	"bytes"
	"github.com/bitfield/script"
)

// This program prints out the names of every second files in the current directory.

func everySecond(in *script.Pipe) *script.Pipe {
	out := bytes.NewBuffer(nil)
	scanner := bufio.NewScanner(in)
	var count = 0
	for scanner.Scan() {
		if count%2 == 0 {
			out.Write(scanner.Bytes())
			out.WriteString("\n")
		}
		count++
	}
	p := script.NewPipe().WithReader(out)
	if err := scanner.Err(); err != nil {
		p = p.WithError(err)
	}
	return p
}

func main() {
	script.ListFiles("*").FilterFunc(everySecond).Stdout()
}
