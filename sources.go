package script

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

// File returns a *Pipe associated with the specified file. This is useful for
// starting pipelines. If there is an error opening the file, the pipe's error
// status will be set.
func File(name string) *Pipe {
	p := NewPipe()
	f, err := os.Open(name)
	if err != nil {
		return p.WithError(err)
	}
	return p.WithReader(f)
}

// Echo returns a pipe containing the supplied string.
func Echo(s string) *Pipe {
	return NewPipe().WithReader(strings.NewReader(s))
}

// Exec runs an external command and returns a pipe containing the output. If
// the command had a non-zero exit status, the pipe's error status will also be
// set to the string "exit status X", where X is the integer exit status.
func Exec(s string) *Pipe {
	return NewPipe().Exec(s)
}

// Stdin returns a pipe which reads from the program's standard input.
func Stdin() *Pipe {
	return NewPipe().WithReader(os.Stdin)
}

// Args creates a pipe containing the program's command-line arguments, one per
// line.
func Args() *Pipe {
	var s strings.Builder
	for _, a := range os.Args[1:] {
		s.WriteString(a + "\n")
	}
	return Echo(s.String())
}

// Get retrives the web resources given by `url` and returns a Pipe containing
// the response body. If an error occurs or a non-200 HTTP Status is encountered
// the error status of the pipe is set appropriately.
func Get(url string) *Pipe {
	p := NewPipe()
	res, err := http.Get(url)
	if err != nil {
		return p.WithError(err)
	}
	if res.StatusCode != 200 {
		return p.WithError(fmt.Errorf("non-200 status: %s", res.Status))
	}
	return p.WithReader(res.Body)
}
