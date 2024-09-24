package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const commandSleep = "sleep"

func main() {
	if len(os.Args) < 2 {
		fmt.Println("No command provided")
		usage()
		os.Exit(1)
	}

	handleSignals()

	switch strings.ToLower(os.Args[1]) {
	case commandSleep:
		if len(os.Args) != 3 {
			fmt.Printf("Usage: %s %s <seconds>\n", os.Args[0], commandSleep)
			usage()
			os.Exit(1)
		}
		err := sleep(os.Args[2])
		if err != nil {
			fmt.Printf("Error sleeping: %s\n", err)
			os.Exit(1)
		}
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Printf("Usage of %s:\n", os.Args[0])
	fmt.Printf("     sleep <seconds>\n")
}

func sleep(seconds string) error {
	s, err := strconv.Atoi(seconds)
	if err != nil {
		return fmt.Errorf("sleep expects an integer, got %s", seconds)
	}
	<-time.After(time.Duration(s) * time.Second)
	return nil
}

func handleSignals() {
	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT)

	go func() {
		<-sigs
		fmt.Println("\nReceived Ctrl+C, exiting...")
		os.Exit(0)
	}()
}
