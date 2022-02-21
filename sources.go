package script

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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

// Exec runs an external command and returns a pipe containing its combined
// output (stdout and stderr).
//
// If the command had a non-zero exit status, the pipe's error status will also
// be set to the string "exit status X", where X is the integer exit status.
//
// For convenience, you can get this value directly as an integer by calling
// ExitStatus on the pipe.
//
// Even in the event of a non-zero exit status, the command's output will still
// be available in the pipe. This is often helpful for debugging. However,
// because String is a no-op if the pipe's error status is set, if you want
// output you will need to reset the error status before calling String.
//
// Note that Exec can also be used as a filter, in which case the given command
// will read from the pipe as its standard input.
func Exec(s string) *Pipe {
	return NewPipe().Exec(s)
}

// File returns a *Pipe that reads from the specified file. This is useful for
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

// FindFiles takes a directory path and returns a pipe listing all the files in
// the directory and its subdirectories recursively, one per line, like Unix
// `find -type f`. If the path doesn't exist or can't be read, the pipe's error
// status will be set.
//
// Each line of the output consists of a slash-separated pathname, starting with
// the initial directory. For example, if the starting directory is "test", and
// it contains 1.txt and 2.txt:
//
// test/1.txt
// test/2.txt
func FindFiles(path string) *Pipe {
	var fileNames []string
	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fileNames = append(fileNames, path)
		}
		return nil
	}
	if err := filepath.Walk(path, walkFn); err != nil {
		return NewPipe().WithError(err)
	}
	return Slice(fileNames)
}

// IfExists tests whether the specified file exists, and returns a pipe whose
// error status reflects the result. If the file doesn't exist, the pipe's error
// status will be set, and if the file does exist, the pipe will have no error
// status. This can be used to do some operation only if a given file exists:
//
// IfExists("/foo/bar").Exec("/usr/bin/something")
func IfExists(filename string) *Pipe {
	_, err := os.Stat(filename)
	if err != nil {
		return NewPipe().WithError(err)
	}
	return NewPipe()
}

// ListFiles creates a pipe containing the files and directories matching the
// supplied path, one per line. The path can be the name of a directory
// (`/path/to/dir`), the name of a file (`/path/to/file`), or a glob (wildcard
// expression) conforming to the syntax accepted by filepath.Match (for example
// `/path/to/*`).
//
// ListFiles does not recurse into subdirectories (use FindFiles for this).
func ListFiles(path string) *Pipe {
	if strings.ContainsAny(path, "[]^*?\\{}!") {
		fileNames, err := filepath.Glob(path)
		if err != nil {
			return NewPipe().WithError(err)
		}
		return Slice(fileNames)
	}
	files, err := ioutil.ReadDir(path)
	if err != nil {
		// Check for the case where the path matches exactly one file
		s, err := os.Stat(path)
		if err != nil {
			return NewPipe().WithError(err)
		}
		if !s.IsDir() {
			return Echo(path)
		}
		return NewPipe().WithError(err)
	}
	fileNames := make([]string, len(files))
	for i, f := range files {
		fileNames[i] = filepath.Join(path, f.Name())
	}
	return Slice(fileNames)
}

// Slice returns a pipe containing each element of the supplied slice of
// strings, one per line.
func Slice(s []string) *Pipe {
	return Echo(strings.Join(s, "\n") + "\n")
}

// Stdin returns a pipe that reads from the program's standard input.
func Stdin() *Pipe {
	return NewPipe().WithReader(os.Stdin)
}
