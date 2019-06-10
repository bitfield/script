package script

import (
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

// Get creates a pipe containing the output of retrieving the web resource
// given by the supplied url.
func Get(url string) *Pipe {
	p := NewPipe()
	res, err := http.Get(url)
	if err != nil {
		return p.WithError(err)
	}
	return p.WithReader(res.Body)
}

// NetCat connnects to the specified address reads the connection until closed
func NetCat(addr string) *Pipe {
	return NewPipe().NetCat(addr)
}
