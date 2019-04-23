package script

import (
	"io"
	"io/ioutil"
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
func (p *Pipe) Error() error {
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
