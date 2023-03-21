package main

import (
	"fmt"
	"os"
)

func main() {
	for i := 1; i <= 3; i++ {
		fmt.Println("This is to stdout")
		fmt.Fprintln(os.Stderr, "This is to stderr")
	}
}
