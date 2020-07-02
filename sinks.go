package script

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

// AppendFile appends the contents of the Pipe to the specified file, and closes
// the pipe after reading. It returns the number of bytes successfully written,
// or an error. If there is an error reading or writing, the pipe's error status
// is also set.
func (p *Pipe) AppendFile(fileName string) (int64, error) {
	return p.writeOrAppendFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY)
}

// Bytes returns the contents of the Pipe as a slice of byte, or an error. If
// there is an error reading, the pipe's error status is also set.
func (p *Pipe) Bytes() ([]byte, error) {
	if p == nil || p.Error() != nil {
		return []byte{}, p.Error()
	}
	res, err := ioutil.ReadAll(p.Reader)
	if err != nil {
		p.SetError(err)
		return []byte{}, err
	}
	return res, nil
}

// CountLines counts lines from the pipe's reader, and returns the integer
// result, or an error. If there is an error reading the pipe, the pipe's error
// status is also set.
func (p *Pipe) CountLines() (int, error) {
	if p == nil || p.Error() != nil {
		return 0, p.Error()
	}
	var lines int
	p.EachLine(func(line string, out *strings.Builder) {
		lines++
	})
	return lines, p.Error()
}

// SHA256Sum calculates the SHA-256 of the file from the pipe's reader, and returns the
// string result, or an error. If there is an error reading the pipe, the pipe's
// error status is also set.
func (p *Pipe) SHA256Sum() (string, error) {
	if p == nil || p.Error() != nil {
		return "", p.Error()
	}

	h := sha256.New()
	if _, err := io.Copy(h, p.Reader); err != nil {
		p.SetError(err)
		return "", p.Error()
	}

	encodedCheckSum := hex.EncodeToString(h.Sum(nil)[:])
	return encodedCheckSum, nil
}

// Slice returns the contents of the pipe as a slice of strings, one element per line, or an error.
// If there is an error reading the pipe, the pipe's error status is also set.
func (p *Pipe) Slice() ([]string, error) {
	if p == nil || p.Error() != nil {
		return nil, p.Error()
	}
	result := []string{}
	p.EachLine(func(line string, out *strings.Builder) {
		result = append(result, line)
	})
	return result, p.Error()
}

// Stdout writes the contents of the pipe to the program's standard output. It
// returns the number of bytes successfully written, plus a non-nil error if the
// write failed or if there was an error reading from the pipe. If the pipe has
// error status, Stdout returns zero plus the existing error.
func (p *Pipe) Stdout() (int, error) {
	if p == nil || p.Error() != nil {
		return 0, p.Error()
	}
	output, err := p.String()
	if err != nil {
		return 0, err
	}
	return fmt.Print(output)
}

// String returns the contents of the Pipe as a string, or an error, and closes
// the pipe after reading. If there is an error reading, the pipe's error status
// is also set.
func (p *Pipe) String() (string, error) {
	data, err := p.Bytes()
	if err != nil {
		p.SetError(err)
		return "", err
	}
	return string(data), nil
}

// WriteFile writes the contents of the Pipe to the specified file, and closes
// the pipe after reading. It returns the number of bytes successfully written,
// or an error. If there is an error reading or writing, the pipe's error status
// is also set.
func (p *Pipe) WriteFile(fileName string) (int64, error) {
	return p.writeOrAppendFile(fileName, os.O_RDWR|os.O_CREATE)
}

func (p *Pipe) writeOrAppendFile(fileName string, mode int) (int64, error) {
	if p == nil || p.Error() != nil {
		return 0, p.Error()
	}
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
