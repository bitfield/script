package script

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
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

// Exec runs an external command and returns a pipe containing the output. If
// the command had a non-zero exit status, the pipe's error status will also be
// set to the string "exit status X", where X is the integer exit status.
func Exec(s string) *Pipe {
	return NewPipe().Exec(s)
}

// IfExists tests whether the specified file exists, and returns a pipe whose
// error status reflects the result. If the file doesn't exist, the pipe's error
// status will be set, and if the file does exist, the pipe will have no error
// status.
func IfExists(filename string) *Pipe {
	_, err := os.Stat(filename)
	if err != nil {
		return NewPipe().WithError(err)
	}
	return NewPipe()
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

// FindFiles takes a directory path and returns a pipe listing all the files in
// the directory and its subdirectories recursively, one per line, like Unix
// `find -type f`. If the path doesn't exist or can't be read, the pipe's error
// status will be set.
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

// ListFiles creates a pipe containing the files and directories matching the
// supplied path, one per line. The path may be a glob, conforming to
// filepath.Match syntax.
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

// Slice returns a pipe containing each element of the supplied slice of strings, one per line.
func Slice(s []string) *Pipe {
	return Echo(strings.Join(s, "\n") + "\n")
}

// Stdin returns a pipe which reads from the program's standard input.
func Stdin() *Pipe {
	return NewPipe().WithReader(os.Stdin)
}

// HTTP executes the given http request with the default HTTP client from the http package. The response of that request,
// is processed by `process`. If process is nil, the default process function copies the body from the request to the pipe.
func HTTP(req *http.Request, process func(*http.Response) (io.Reader, error)) *Pipe {
	return HTTPWithClient(http.DefaultClient, req, process)
}

// HTTPClient is an interface to allow the user to plugin alternative HTTP clients into the source.
// The HTTPClient interface is a subset of the methods provided by the http.Client
// We use an own interface with a minimal surface to allow make it easy to implement own customized clients.
type HTTPClient interface {
	Do(r *http.Request) (*http.Response, error)
}

// HTTP executes the given http request with the given HTTPClient. The response of that request,
// is processed by `process`. If process is nil, the default process function copies the body from the request to the pipe.
func HTTPWithClient(client HTTPClient, req *http.Request, process func(*http.Response) (io.Reader, error)) *Pipe {
	p := NewPipe()
	if req == nil {
		p.SetError(errors.New("no request specified"))
		return p
	}
	if process == nil {
		process = defaultHTTPProcessor
	}
	resp, err := client.Do(req)
	if err != nil {
		p.SetError(err)
		return p
	}
	reader, err := process(resp)
	return p.WithReader(reader).WithError(err)
}

// defaultHTTPProcessor returns the response body as reader if there is a body in the response. Otherwise it will return a
// reader with the empty string to simulate an empty body.
func defaultHTTPProcessor(resp *http.Response) (io.Reader, error) {
	if resp.Body != nil {
		return resp.Body, nil
	}
	return bytes.NewBufferString(""), nil
}

// AssertingHTTPProcessor is an HTTP processor checking if the HTTP response has the expected code. If the code is not the
// expected code an error is returned. Otherwise the body of the response is returned as a reader.
func AssertingHTTPProcessor(code int) func(*http.Response) (io.Reader, error) {
	return func(resp *http.Response) (io.Reader, error) {
		if resp.StatusCode != code {
			return bytes.NewBufferString(""), fmt.Errorf("got HTTP status code %d instead of expected %d", resp.StatusCode, code)
		}
		return defaultHTTPProcessor(resp)
	}
}
