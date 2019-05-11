package main

import "github.com/bitfield/script"

// The filter form of cat copies its standard input to its standard output.

func main() {
	script.Stdin().Stdout()
}
