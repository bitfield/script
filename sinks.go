package script

import (
	"bufio"
	"io/ioutil"
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
