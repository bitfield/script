package main

import (
	"github.com/rmasci/script"
)

func main() {
	script.Echo("one two three four five").Column(1).Stdout()
	script.Echo("one, two, three, four, five").Columns("; ", "|", 3, 2, 1, 5, 4).Stdout()
	script.Echo("one, two, three, four, five").Columns(", ", "|", 3, 6, 57, 2, 1, 5, 4).Stdout()

}
