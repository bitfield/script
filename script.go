// Package script is a collection of utilities for doing the kind of tasks that
// shell scripts are good at. It provides equivalent for Unix `cat`, `wc`, and
// so on. It also provides a 'pipeline' construct which allows you to chain all
// these operations together, just like shells do.
package script

import (
	"bufio"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

type pipe struct {
	reader io.Reader
	close  bool
}

// Pipe returns a new Pipe with no associated reader.
func Pipe() pipe {
	return pipe{nil, false}
}

// WithReader takes an io.Reader and returns a pipe associated with that reader,
// and its `close` flag set to false, to indicate that the pipe does not need
// closing after reading.
func (p pipe) WithReader(r io.Reader) pipe {
	return pipe{r, false}
}

// WithCloser takes an io.ReadCloser and returns a pipe associated with that
// source, and its `close` flag set to true, to indicate that the pipe needs to
// be closed after reading.
func (p pipe) WithCloser(r io.ReadCloser) pipe {
	return pipe{r.(io.Reader), true}
}

// CloseIfNecessary will close the reader associated with the pipe, if it needs
// closing.
func (p pipe) CloseIfNecessary() {
	if p.close {
		p.reader.(io.Closer).Close()
	}
}

// String returns the contents of the pipe as a string.
func (p pipe) String() string {
	res, err := ioutil.ReadAll(p.reader)
	if err != nil {
		log.Fatal(err)
	}
	p.CloseIfNecessary()
	return string(res)
}

// Int returns the contents of the pipe as an integer.
func (p pipe) Int() int {
	res, err := strconv.Atoi(p.String())
	if err != nil {
		log.Fatal(err)
	}
	return res
}

// File returns a pipe full of the contents of the specified file. This is useful
// for starting pipelines.
func File(name string) pipe {
	r, err := os.Open(name)
	if err != nil {
		log.Fatal(err)
	}
	return Pipe().WithCloser(r)
}

// CountLines counts lines in the specified file and returns the integer result.
func CountLines(name string) int {
	return File(name).CountLines()
}

// CountLines counts lines in its input and returns the integer result.
func (p pipe) CountLines() int {
	scanner := bufio.NewScanner(p.reader)
	var lines int
	for scanner.Scan() {
		lines++
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	p.CloseIfNecessary()
	return lines
}
