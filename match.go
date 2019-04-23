package script

import (
	"bufio"
	"strings"
)

// Match reads from the pipe, and returns a new pipe containing only lines which
// contain the specified string. If there is an error reading the pipe, the
// pipe's error status is also set.
func (p Pipe) Match(s string) *Pipe {
	if p.Error() != nil {
		return &p
	}
	scanner := bufio.NewScanner(p.Reader)
	output := strings.Builder{}
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), s) {
			output.WriteString(scanner.Text())
			output.WriteByte('\n')
		}
	}
	err := scanner.Err()
	if err != nil {
		p.SetError(err)
	}
	p.Close()
	return Echo(output.String())
}

// Reject reads from the pipe, and returns a new pipe containing only lines
// which do not contain the specified string. If there is an error reading the
// pipe, the pipe's error status is also set.
func (p Pipe) Reject(s string) *Pipe {
	if p.Error() != nil {
		return &p
	}
	scanner := bufio.NewScanner(p.Reader)
	output := strings.Builder{}
	for scanner.Scan() {
		if !strings.Contains(scanner.Text(), s) {
			output.WriteString(scanner.Text())
			output.WriteByte('\n')
		}
	}
	err := scanner.Err()
	if err != nil {
		p.SetError(err)
	}
	p.Close()
	return Echo(output.String())
}
