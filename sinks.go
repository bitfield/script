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

// CountLines counts lines from the pipe's reader, and returns the integer
// result, or an error. If there is an error reading the pipe, the pipe's error
// status is also set.
func (p *Pipe) CountLines() (int, error) {
	var lines int
	p.EachLine(func(line string, out *strings.Builder) {
		lines++
	})
	return lines, p.Error()
}
