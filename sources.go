package script

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
)

// File returns a *Pipe associated with the specified file. This is useful for
// starting pipelines. If there is an error opening the file, the pipe's error
// status will be set.
func File(name string) *Pipe {
	r, err := os.Open(name)
	if err != nil {
		return NewPipe().WithError(err)
	}
	return NewPipe().WithCloser(r)
}

// Echo returns a pipe containing the supplied string.
func Echo(s string) *Pipe {
	return NewPipe().WithReader(strings.NewReader(s))
}

// Exec runs an external command and returns a pipe containing the output. If
// the command had a non-zero exit status, the pipe's error status will also be
// set to the string "exit status X", where X is the integer exit status.
func Exec(s string) *Pipe {
	p := NewPipe()
	args := strings.Fields(s)
	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		p.SetError(err)
	}
	return p.WithReader(bytes.NewReader(output))
}

// Stdin returns a pipe which reads from the program's standard input.
func Stdin() *Pipe {
	return NewPipe().WithReader(os.Stdin)
}
