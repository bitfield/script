package script

import (
	"io"
	"io/ioutil"
)

// ReadAutoCloser wraps an io.Reader, and closes it automatically, if closable,
// once it has been completely read.
type ReadAutoCloser struct {
	r io.Reader
}

// Read reads up to len(p) bytes from the data source into p. It returns the
// number of bytes read and any error encountered. At end of file, Read returns
// 0, io.EOF. In the EOF case, the data source will be closed.
func (a ReadAutoCloser) Read(b []byte) (n int, err error) {
	if a.r == nil {
		return 0, io.EOF
	}
	n, err = a.r.Read(b)
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
	return ReadAutoCloser{r}
}
