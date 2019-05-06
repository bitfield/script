package script

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	switch os.Getenv("SCRIPT_TEST") {
	case "args":
		// Print out command-line arguments
		Args().Stdout()
	default:
		os.Exit(m.Run())
	}
}
