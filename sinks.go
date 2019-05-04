package script

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
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

// WriteFile writes the contents of the Pipe to the specified file, and closes
// the pipe after reading. It returns the number of bytes successfully written,
// or an error. If there is an error reading or writing, the pipe's error status
// is also set.
func (p *Pipe) WriteFile(fileName string) (int64, error) {
	return p.writeOrAppendFile(fileName, os.O_RDWR|os.O_CREATE)
}

// AppendFile appends the contents of the Pipe to the specified file, and closes
// the pipe after reading. It returns the number of bytes successfully written,
// or an error. If there is an error reading or writing, the pipe's error status
// is also set.
func (p *Pipe) AppendFile(fileName string) (int64, error) {
	return p.writeOrAppendFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY)
}

func (p *Pipe) writeOrAppendFile(fileName string, mode int) (int64, error) {
	if p.Error() != nil {
		return 0, p.Error()
	}
	defer p.Close()
	out, err := os.OpenFile(fileName, mode, 0644)
	if err != nil {
		p.SetError(err)
		return 0, err
	}
	defer out.Close()
	wrote, err := io.Copy(out, p.Reader)
	if err != nil {
		p.SetError(err)
		return 0, err
	}
	return wrote, nil
}

// Stdout writes the contents of the pipe to the program's standard output. It
// returns the number of bytes successfully written, plus a non-nil error if the
// write failed or if there was an error reading from the pipe. If the pipe has
// error status, Stdout returns zero plus the existing error.
func (p *Pipe) Stdout() (int, error) {
	if p.Error() != nil {
		return 0, p.Error()
	}
	defer p.Close()
	output, err := p.String()
	if err != nil {
		return 0, err
	}
	return fmt.Print(output)
}
