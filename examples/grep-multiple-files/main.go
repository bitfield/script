/*
We can grep word `hello` in multiple files inside directory
*/
package main

import (
	"github.com/bitfield/script"
)

func main() {
	script.ListFiles("data/").Concat().Match("hello").Stdout()
}
