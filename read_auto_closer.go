package script

import (
	"io"
	"io/ioutil"
)

// ReadAutoCloser represents a pipe source which will be automatically closed
// once it has been fully read.
type ReadAutoCloser struct {
	r io.ReadCloser
}

// Read reads up to len(buf) bytes from the data source into buf. It returns the
// number of bytes read and any error encountered. At end of file, Read returns
// 0, io.EOF. In the EOF case, the data source will be closed.
func (a ReadAutoCloser) Read(buf []byte) (n int, err error) {
	if a.r == nil {
		return 0, io.EOF
	}
	n, err = a.r.Read(buf)
	if err == io.EOF {
		a.Close()
	}
	return n, err
}

// Close closes the data source associated with a, and returns the result of
// that close operation.
func (a ReadAutoCloser) Close() error {
	if a.r == nil {
		return nil
	}
	return a.r.(io.Closer).Close()
}

// NewReadAutoCloser returns an ReadAutoCloser wrapping the supplied Reader. If
// the Reader is not a Closer, it will be wrapped in a NopCloser to make it
// closable.
func NewReadAutoCloser(r io.Reader) ReadAutoCloser {
	if _, ok := r.(io.Closer); !ok {
		return ReadAutoCloser{ioutil.NopCloser(r)}
	}
	rc, ok := r.(io.ReadCloser)
	if !ok {
		// This can never happen, but just in case it does...
		panic("internal error: type assertion to io.ReadCloser failed")
	}
	return ReadAutoCloser{rc}
}
