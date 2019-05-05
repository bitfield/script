package script

import (
	"io"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
)

// Pipe represents a pipe object with an associated Reader.
type Pipe struct {
	Reader io.ReadCloser
	err    error
}

// NewPipe returns a pointer to a new empty pipe.
func NewPipe() *Pipe {
	return &Pipe{ioutil.NopCloser(strings.NewReader("")), nil}
}

// Close closes the pipe's associated reader. This is always safe to do, because
// pipes created from a non-closable source will have an `ioutil.NopCloser` to
// call.
func (p *Pipe) Close() error {
	if p == nil || p.Reader == nil {
		return nil
	}
	return p.Reader.Close()
}

// Error returns the last error returned by any pipe operation, or nil otherwise.
func (p *Pipe) Error() error {
	if p == nil {
		return nil
	}
	return p.err
}

var exitStatusPattern = regexp.MustCompile(`exit status (\d+)$`)

// ExitStatus returns the integer exit status of a previous command, if the
// pipe's error status is set, and if the error matches the pattern "exit status
// %d". Otherwise, it returns zero.
func (p *Pipe) ExitStatus() int {
	if p.Error() == nil {
		return 0
	}
	match := exitStatusPattern.FindStringSubmatch(p.Error().Error())
	if len(match) < 2 {
		return 0
	}
	status, err := strconv.Atoi(match[1])
	if err != nil {
		// This seems unlikely, but...
		return 0
	}
	return status
}

// WithReader takes an io.Reader which does not need to be closed after reading,
// and associates the pipe with that reader.
func (p *Pipe) WithReader(r io.Reader) *Pipe {
	return p.WithCloser(ioutil.NopCloser(r))
}

// WithCloser takes an io.ReadCloser and associates the pipe with that source.
func (p *Pipe) WithCloser(r io.ReadCloser) *Pipe {
	if p == nil || p.Reader == nil {
		return nil
	}
	p.Reader = r
	return p
}

// SetError sets the pipe's error status to the specified error.
func (p *Pipe) SetError(err error) {
	if p != nil {
		p.err = err
	}
}

// WithError sets the pipe's error status to the specified error and returns the
// modified pipe.
func (p *Pipe) WithError(err error) *Pipe {
	p.SetError(err)
	return p
}
