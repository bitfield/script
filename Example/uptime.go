package main

import "github.com/rmasci/script"

func main() {
	script.Exec("uptime").Columns(" ", ",", 12, 11, 10).Stdout()
}
