package script

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// HTTP executes the given http request with the default HTTP client from the http package. The response of that request,
// is processed by `process`. If process is nil, the default process function copies the body from the request to the pipe.
func HTTP(url string) *HTTPPipe {
	exec := newHTTPExecutor()
	var err error
	exec.request, err = http.NewRequest(http.MethodGet, url, nil)
	p := NewPipe().WithReader(exec).WithError(err)
	return &HTTPPipe{
		*p,
		exec,
	}
}

// HTTP creates a new HTTPPipe from the given Pipe. It acts as sink
func (p *Pipe) HTTP(url string) *HTTPPipe {
	exec := newHTTPExecutor()
	var err error
	exec.request, err = http.NewRequest(http.MethodPost, url, p.Reader)
	if err != nil {
		p.SetError(err)
	}
	p = p.WithReader(exec)
	return &HTTPPipe{
		*p,
		exec,
	}
}

func (h *HTTPPipe) Error() error {
	if h.err != nil {
		return h.err
	}

	return h.executor.doRequest()
}

func (h *HTTPPipe) WithMethod(method string) *HTTPPipe {
	if h != nil && h.executor != nil && h.executor.request != nil {
		h.executor.request.Method = method
		return h
	}
	return nil
}

func (h *HTTPPipe) WithBody(body io.ReadCloser) *HTTPPipe {
	if h != nil && h.executor != nil && h.executor.request != nil {
		h.executor.request.Body = body
		return h
	}
	return nil
}

func (h *HTTPPipe) WithHeader(header http.Header) *HTTPPipe {
	if h != nil && h.executor != nil && h.executor.request != nil {
		h.executor.request.Header = header
		return h
	}
	return nil
}

func (h *HTTPPipe) WithClient(client HTTPClient) *HTTPPipe {
	if h == nil || h.executor == nil {
		return nil
	}
	h.executor.client = client
	return h
}

type HTTPPipe struct {
	Pipe
	executor *httpExecutor
}

type ResponseProcessor func(res *http.Response) (io.Reader, error)

type httpExecutor struct {
	executed       bool
	client         HTTPClient
	request        *http.Request
	responseReader io.Reader
	processor      ResponseProcessor
}

func newHTTPExecutor() *httpExecutor {
	return &httpExecutor{
		client:    http.DefaultClient,
		executed:  false,
		processor: defaultHTTPProcessor,
	}
}

func (h *httpExecutor) doRequest() (err error) {
	if h.executed {
		return nil
	}
	defer func() { h.executed = true }()
	if h.request == nil {
		return errors.New("there is no request set")
	}
	resp, err := h.client.Do(h.request)
	if err != nil {
		return err
	}
	h.responseReader, err = h.processor(resp)
	return err
}

func (h *httpExecutor) Read(p []byte) (n int, err error) {
	if !h.executed {
		err = h.doRequest()
		if err != nil {
			return 0, err
		}
	}
	return h.responseReader.Read(p)
}

func (h *httpExecutor) Close() error {
	if closer, ok := h.responseReader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// HTTPClient is an interface to allow the user to plugin alternative HTTP clients into the source.
// The HTTPClient interface is a subset of the methods provided by the http.Client
// We use an own interface with a minimal surface to allow make it easy to implement own customized clients.
type HTTPClient interface {
	Do(r *http.Request) (*http.Response, error)
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
