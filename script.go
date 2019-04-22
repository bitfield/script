// Package script aims to make it easy to write shell-type scripts in Go, for general system administration purposes: reading files, counting lines, matching strings, and so on.
package script

import (
	"bufio"
	"io"
	"io/ioutil"
	"os"
)

// Pipe represents a pipe object with an associated Reader.
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

// SetError sets the pipe's error status to the specified error.
func (p *Pipe) SetError(err error) {
	p.err = err
}

// WithError sets the pipe's error status to the specified error and returns the
// modified pipe.
func (p *Pipe) WithError(err error) *Pipe {
	p.SetError(err)
	return p
}

// String returns the contents of the Pipe as a string, or an error, and closes the pipe after reading. If there is an error reading, the
// pipe's error status is also set.
func (p *Pipe) String() (string, error) {
	if p.Error() != nil {
		return "", p.Error()
	}
	res, err := ioutil.ReadAll(p.Reader)
	if err != nil {
		p.SetError(err)
		return "", err
	}
	p.Close()
	return string(res), nil
}

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

// CountLines counts lines in the specified file and returns the integer result,
// or an error, and closes the pipe after reading. If there is an error reading
// the pipe, the pipe's error status is also set.
func CountLines(name string) (int, error) {
	return File(name).CountLines()
}

// CountLines counts lines from the pipe's reader, and returns the integer
// result, or an error. If there is an error reading the pipe, the pipe's error
// status is also set.
func (p *Pipe) CountLines() (int, error) {
	if p.Error() != nil {
		return 0, p.Error()
	}
	scanner := bufio.NewScanner(p.Reader)
	var lines int
	for scanner.Scan() {
		lines++
	}
	err := scanner.Err()
	if err != nil {
		p.SetError(err)
	}
	p.Close()
	return lines, err
}
