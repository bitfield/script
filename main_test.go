package script_test

import (
	"os"
	"testing"

	"github.com/bitfield/script"
)

func TestMain(m *testing.M) {
	switch os.Getenv("SCRIPT_TEST") {
	case "args":
		// Print out command-line arguments
		script.Args().Stdout()
	case "stdin":
		// Echo input to output
		script.Stdin().Stdout()
	default:
		os.Exit(m.Run())
	}
}
