package main

import (
	"fmt"
	"github.com/bitfield/script"
)

func main() {
	files, _ := script.ListFiles(".").String()
	fmt.Println(files)

	files, _ = script.ListFiles("../../test?ata/*").String()
	fmt.Println(files)
}
