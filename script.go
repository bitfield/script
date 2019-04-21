// Package script is a collection of utilities for doing the kind of tasks that
// shell scripts are good at. It provides equivalent for Unix `cat`, `wc`, and
// so on. It also provides a 'Pipe' construct which allows you to chain all
// these operations together, just like shells do.
package script

import (
	"bufio"
	"io"
	"io/ioutil"
	"log"
	"os"
)

// Pipe represents a pipe object with an associated reader.
type Pipe struct {
	Reader io.ReadCloser
}

// Close closes the pipe's associated reader.
func (p Pipe) Close() error {
	return p.Reader.Close()
}

// WithReader takes an io.Reader which does not need to be closed after reading,
// and returns a new Pipe associated with that reader.
func (Pipe) WithReader(r io.Reader) Pipe {
	return Pipe{ioutil.NopCloser(r)}
}

// WithCloser takes an io.ReadCloser and returns a Pipe associated with that
// source.
func (Pipe) WithCloser(r io.ReadCloser) Pipe {
	return Pipe{r}
}

// String returns the contents of the Pipe as a string.
func (p Pipe) String() string {
	res, err := ioutil.ReadAll(p.Reader)
	if err != nil {
		log.Fatal(err)
	}
	p.Close()
	return string(res)
}

// File returns a Pipe full of the contents of the specified file. This is useful
// for starting Pipelines.
func File(name string) Pipe {
	r, err := os.Open(name)
	if err != nil {
		log.Fatal(err)
	}
	return Pipe{}.WithCloser(r)
}

// CountLines counts lines in the specified file and returns the integer result.
func CountLines(name string) int {
	return File(name).CountLines()
}

// CountLines counts lines in its input and returns the integer result.
func (p Pipe) CountLines() int {
	scanner := bufio.NewScanner(p.Reader)
	var lines int
	for scanner.Scan() {
		lines++
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	p.Close()
	return lines
}
