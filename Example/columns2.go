package main

import (
	"github.com/rmasci/script"
)

func main() {
	p := script.File("one2ten.csv").Columns(",", ",", "7, 5, 2, 6, 8:")
	p.Stdout()
}
