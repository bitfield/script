package main

import (
	"fmt"

	"github.com/rmasci/script"
)

func main() {
	str, _ := script.Exec("whoami").TrimSpace().String()
	str2, _ := script.Exec("whoami").String()
	fmt.Println(str, "is my username")
	fmt.Println(str2, "is my username")
}
