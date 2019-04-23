package script

import (
	"os"
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
