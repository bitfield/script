package script

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// Match reads from the pipe, and returns a new pipe containing only lines which
// contain the specified string. If there is an error reading the pipe, the
// pipe's error status is also set.
func (p *Pipe) Match(s string) *Pipe {
	return p.EachLine(func(line string, out *strings.Builder) {
		if strings.Contains(line, s) {
			out.WriteString(line)
			out.WriteByte('\n')
		}
	})
}

// MatchRegexp reads from the pipe, and returns a new pipe containing only lines
// which match the specified compiled regular expression. If there is an error
// reading the pipe, the pipe's error status is also set.
func (p *Pipe) MatchRegexp(re *regexp.Regexp) *Pipe {
	return p.EachLine(func(line string, out *strings.Builder) {
		if re.MatchString(line) {
			out.WriteString(line)
			out.WriteByte('\n')
		}
	})
}

// Reject reads from the pipe, and returns a new pipe containing only lines
// which do not contain the specified string. If there is an error reading the
// pipe, the pipe's error status is also set.
func (p *Pipe) Reject(s string) *Pipe {
	return p.EachLine(func(line string, out *strings.Builder) {
		if !strings.Contains(line, s) {
			out.WriteString(line)
			out.WriteByte('\n')
		}
	})
}

// RejectRegexp reads from the pipe, and returns a new pipe containing only
// lines which don't match the specified compiled regular expression. If there
// is an error reading the pipe, the pipe's error status is also set.
func (p *Pipe) RejectRegexp(re *regexp.Regexp) *Pipe {
	return p.EachLine(func(line string, out *strings.Builder) {
		if !re.MatchString(line) {
			out.WriteString(line)
			out.WriteByte('\n')
		}
	})
}

// EachLine calls the specified function for each line of input, passing it the
// line as a string, and a *strings.Builder to write its output to. The return
// value from EachLine is a pipe containing the contents of the strings.Builder.
func (p *Pipe) EachLine(process func(string, *strings.Builder)) *Pipe {
	if p == nil || p.Error() != nil {
		return p
	}
	scanner := bufio.NewScanner(p.Reader)
	output := strings.Builder{}
	for scanner.Scan() {
		process(scanner.Text(), &output)
	}
	err := scanner.Err()
	if err != nil {
		p.SetError(err)
	}
	return Echo(output.String())
}

// Exec runs an external command and returns a pipe containing the output. If
// the command had a non-zero exit status, the pipe's error status will also be
// set to the string "exit status X", where X is the integer exit status.
func (p *Pipe) Exec(s string) *Pipe {
	if p == nil || p.Error() != nil {
		return p
	}
	q := NewPipe()
	args := strings.Fields(s)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = p.Reader
	output, err := cmd.CombinedOutput()
	if err != nil {
		q.SetError(err)
	}
	return q.WithReader(bytes.NewReader(output))
}

// ExecAt runs an external command at the specified directory and returns a pipe
// containing the output. If the command had a non-zero exit status, the pipe's
// error status will also be set to the string "exit status X", where X is the
// integer exit status.
func (p *Pipe) ExecAt(dir, s string) *Pipe {
	if p == nil || p.Error() != nil {
		return p
	}
	q := NewPipe()
	args := strings.Fields(s)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = p.Reader
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		q.SetError(err)
	}
	return q.WithReader(bytes.NewReader(output))
}

// Join reads the contents of the pipe, line by line, and joins them into a
// single space-separated string. It returns a pipe containing this string. Any
// terminating newline is preserved.
func (p *Pipe) Join() *Pipe {
	if p == nil || p.Error() != nil {
		return p
	}
	result, err := p.String()
	if err != nil {
		return p
	}
	var terminator string
	if strings.HasSuffix(result, "\n") {
		terminator = "\n"
		result = result[:len(result)-1]
	}
	output := strings.ReplaceAll(result, "\n", " ")
	return Echo(output + terminator)
}

// Concat reads a list of filenames from the pipe, one per line, and returns a
// pipe which reads all those files in sequence. If there are any errors (for
// example, non-existent files), the pipe's error status will be set to the
// first error encountered, but execution will continue.
func (p *Pipe) Concat() *Pipe {
	if p == nil || p.Error() != nil {
		return p
	}
	var readers []io.Reader
	p.EachLine(func(line string, out *strings.Builder) {
		input, err := os.Open(line)
		if err != nil {
			p.SetError(err)
			return
		}
		readers = append(readers, NewReadAutoCloser(input))
	})
	return p.WithReader(io.MultiReader(readers...))
}
