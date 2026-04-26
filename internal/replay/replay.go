package replay

import (
	"bytes"
	"io"
	"net/http"
)

type CapturedRequest struct {
	Method string
	Path   string
	Query  string
	Header http.Header
	Body   []byte
}

func Capture(r *http.Request) (CapturedRequest, error) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return CapturedRequest{}, err
	}
	r.Body = io.NopCloser(bytes.NewReader(b))
	return CapturedRequest{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Header: r.Header.Clone(), Body: b}, nil
}

func (c CapturedRequest) NewRequest(baseURL string) (*http.Request, error) {
	u := baseURL + c.Path
	if c.Query != "" {
		u += "?" + c.Query
	}
	req, err := http.NewRequest(c.Method, u, bytes.NewReader(c.Body))
	if err != nil {
		return nil, err
	}
	req.Header = c.Header.Clone()
	return req, nil
}
