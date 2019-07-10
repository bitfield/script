package script

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

const (
	procDir = "/proc"
)

// Args creates a pipe containing the program's command-line arguments, one per
// line.
func Args() *Pipe {
	var s strings.Builder
	for _, a := range os.Args[1:] {
		s.WriteString(a + "\n")
	}
	return Echo(s.String())
}

// Echo returns a pipe containing the supplied string.
func Echo(s string) *Pipe {
	return NewPipe().WithReader(strings.NewReader(s))
}

// Exec runs an external command and returns a pipe containing the output. If
// the command had a non-zero exit status, the pipe's error status will also be
// set to the string "exit status X", where X is the integer exit status.
func Exec(s string) *Pipe {
	return NewPipe().Exec(s)
}

// File returns a *Pipe associated with the specified file. This is useful for
// starting pipelines. If there is an error opening the file, the pipe's error
// status will be set.
func File(name string) *Pipe {
	p := NewPipe()
	f, err := os.Open(name)
	if err != nil {
		return p.WithError(err)
	}
	return p.WithReader(f)
}

// Stdin returns a pipe which reads from the program's standard input.
func Stdin() *Pipe {
	return NewPipe().WithReader(os.Stdin)
}

//Processes reads /proc directory for info about processes
//It prints line by line pid and cmdline of each processes
func Processes() *Pipe {
	p := NewPipe()
	var s strings.Builder
	files, err := ioutil.ReadDir(procDir)
	if err != nil {
		return p.WithError(err)
	}
	for _, f := range files {
		pid, err := strconv.Atoi(f.Name())
		if err != nil {
			continue
		}
		cmdlineFile := fmt.Sprintf("%s/%d/%s", procDir, pid, "cmdline")
		data, err := ioutil.ReadFile(cmdlineFile)
		// Here we get only first word for cmdline
		cmdline := string(bytes.Split(data, []byte{byte(0)})[0])
		s.WriteString(fmt.Sprintf("%d %s\n", pid, cmdline))
	}
	return Echo(s.String())
}
