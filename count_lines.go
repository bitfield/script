package script

import (
	"bufio"
)

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
