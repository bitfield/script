package script

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

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

// ListFiles returns a *Pipe containing list of files under the given path or glob. This can
// be used to iterate over the files. If there is an error with the path or the pattern of glob, the pipe's error
// status will be set.
func ListFiles(path string) *Pipe {
	var matches []string
	p := NewPipe()
	if strings.ContainsAny(path, "[]^*?\\{}!") {
		var err error
		matches, err = filepath.Glob(path)
		if err != nil {
			return p.WithError(err)
		}
	} else {
		files, err := ioutil.ReadDir(path)
		if err != nil {
			return p.WithError(err)
		}
		for _, file := range files {
			matches = append(matches, path+"/"+file.Name())
		}
	}
	return NewPipe().WithReader(strings.NewReader(strings.Join(matches, "\n")))
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

// Stdin returns a pipe which reads from the program's standard input.
func Stdin() *Pipe {
	return NewPipe().WithReader(os.Stdin)
}

// Args creates a pipe containing the program's command-line arguments, one per
// line.
func Args() *Pipe {
	var s strings.Builder
	for _, a := range os.Args[1:] {
		s.WriteString(a + "\n")
	}
	return Echo(s.String())
}
