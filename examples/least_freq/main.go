package main

import "github.com/bitfield/script"

// filters the least frequent line from input and copies it to standard output

func main() {
	script.Stdin().Freq().Last(1).Stdout()
}
