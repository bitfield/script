package script

import (
	"io/ioutil"
	"strings"
)

// String returns the contents of the Pipe as a string, or an error, and closes the pipe after reading. If there is an error reading, the
// pipe's error status is also set.
func (p *Pipe) String() (string, error) {
	if p.Error() != nil {
		return "", p.Error()
	}
	defer p.Close()
	res, err := ioutil.ReadAll(p.Reader)
	if err != nil {
		p.SetError(err)
		return "", err
	}
	return string(res), nil
}

// Echo returns a pipe containing the supplied string.
func Echo(s string) *Pipe {
	return NewPipe().WithReader(strings.NewReader(s))
}
