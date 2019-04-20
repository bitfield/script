// Package script is a collection of utilities for doing the kind of tasks that
// shell scripts are good at. It provides equivalent for Unix `cat`, `wc`, and
// so on. It also provides a 'pipeline' construct which allows you to chain all
// these operations together, just like shells do.
package script

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

// Pipe represents a pipe, which allows various operations to be chained
// together. Most operations either return a pipe, or are methods on a pipe, or
// both, so we can chain calls like this:
//
//      result := Cat("foo").CountLines().String()
//
// If `close` is true, the pipe needs to be closed after reading (for example, a
// file).
type Pipe struct {
	Reader io.Reader
	close  bool
}

// Echo returns a pipe full of the specified string. This is useful for starting
// pipelines.
func Echo(s string) Pipe {
	reader := bytes.NewReader([]byte(s))
	return Pipe{reader, false}
}

// String returns the contents of the pipe as a string.
func (p Pipe) String() string {
	res, err := ioutil.ReadAll(p.Reader)
	if err != nil {
		log.Fatal(err)
	}
	if p.close {
		p.Reader.(io.Closer).Close()
	}
	return string(res)
}

// Int returns the contents of the pipe as an integer.
func (p Pipe) Int() int {
	res, err := strconv.Atoi(p.String())
	if err != nil {
		log.Fatal(err)
	}
	return (res)
}

// Cat on a pipe is a no-op, returning a pipe full of the contents of the pipe.
func (p Pipe) Cat() Pipe {
	return p
}

// Cat returns a pipe full of the contents of the specified file. This is useful
// for starting pipelines.
func Cat(name string) Pipe {
	out, err := os.Open(name)
	if err != nil {
		log.Fatal(err)
	}
	return Pipe{out, true}
}

// CountLines counts lines in the specified file and returns a pipe full of the
// integer result.
func CountLines(name string) Pipe {
	return Cat(name).CountLines()
}

// CountLines counts lines in its input and returns a pipe full of the integer
// result.
func (p Pipe) CountLines() Pipe {
	scanner := bufio.NewScanner(p.Reader)
	var lines int
	for scanner.Scan() {
		lines++
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return Echo(fmt.Sprintf("%d", lines))
}
