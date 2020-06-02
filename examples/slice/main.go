package main

import (
	"fmt"
	"github.com/bitfield/script"
	"log"
)

func main() {
	// Get a Slice with all the running process PID
	pids, err := script.Exec("ps -ea").Column(1).Slice()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Running process PIDs are:")
	for _, pid := range pids{
		fmt.Println(pid)
	}
}
