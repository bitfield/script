// Package script is a collection of utilities for doing the kind of tasks that
// shell scripts are good at: reading files, counting lines, matching strings,
// and so on.
//
// Most operations return a Pipe object, so that operations can be chained:
//
//      res := File("test.txt").CountLines()
//
// If any pipe operation results in an error, the pipe's Error() method will
// return that error, and all pipe operations will be no-ops. Thus you can
// safely chain a whole series of operations without having to check the error
// status at each stage:
//
//      p := File("doesnt_exist.txt")
//      out := p.String() // succeeds, with empty result
//      res := p.CountLines() // succeeds, with zero result
//      fmt.Println(p.Error())
//
// Output: open doesnt_exist.txt: no such file or directory
package script

import (
	"bufio"
	"io"
	"io/ioutil"
	"os"
)

// Pipe represents a pipe object with an associated reader.
type Pipe struct {
	Reader io.ReadCloser
	err    error
}

// NewPipe returns a pointer to a new empty pipe.
func NewPipe() *Pipe {
	return &Pipe{nil, nil}
}

// Close closes the pipe's associated reader. This is always safe to do, because
// pipes created from a non-closable source will have an `ioutil.NopCloser` to
// call.
func (p *Pipe) Close() error {
	return p.Reader.Close()
}

// Error returns the last error returned by any pipe operation, or nil otherwise.
func (p Pipe) Error() error {
	return p.err
}

// WithReader takes an io.Reader which does not need to be closed after reading,
// and associates the pipe with that reader.
func (p *Pipe) WithReader(r io.Reader) *Pipe {
	return p.WithCloser(ioutil.NopCloser(r))
}

// WithCloser takes an io.ReadCloser and associates the pipe with that source.
func (p *Pipe) WithCloser(r io.ReadCloser) *Pipe {
	p.Reader = r
	return p
}

// WithError sets the pipe's error status to the specified error.
func (p *Pipe) WithError(err error) *Pipe {
	p.err = err
	return p
}

// String returns the contents of the Pipe as a string. As with all pipe-reading
// operations, this returns the zero value if the pipe has a non-nil error
// status, and closes the pipe after reading. If there is an error reading, the
// pipe's error status will be set.
func (p *Pipe) String() string {
	if p.Error() != nil {
		return ""
	}
	res, err := ioutil.ReadAll(p.Reader)
	if err != nil {
		p.err = err
		return ""
	}
	p.Close()
	return string(res)
}

// File returns a Pipe associated with the specified file. This is useful for
// starting pipelines. If there is an error opening the file, the pipe's error
// status will be set.
func File(name string) *Pipe {
	r, err := os.Open(name)
	if err != nil {
		return NewPipe().WithError(err)
	}
	return NewPipe().WithCloser(r)
}

// CountLines counts lines in the specified file and returns the integer result.
// As with all pipe-reading operations, this returns the zero value if the pipe
// has a non-nil error status, and closes the pipe after reading.
func CountLines(name string) int {
	return File(name).CountLines()
}

// CountLines counts lines from the pipe's reader, and returns the integer
// result. As with all pipe-reading operations, this returns the zero value if
// the pipe has a non-nil error status, and closes the pipe after reading. If
// there is an error reading the pipe, the pipe's error status is set.
func (p *Pipe) CountLines() int {
	if p.Error() != nil {
		return 0
	}
	scanner := bufio.NewScanner(p.Reader)
	var lines int
	for scanner.Scan() {
		lines++
	}
	if err := scanner.Err(); err != nil {
		p.err = err
	}
	p.Close()
	return lines
}
