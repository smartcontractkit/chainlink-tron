package httpclient

import (
	"bytes"
	"io"
	"net/http"
)

type MockHttpClient struct {
	code     int
	response []byte
	error    error
}

func NewMockHttpClient(code int, response []byte, err error) *MockHttpClient {
	return &MockHttpClient{
		code:     code,
		response: response,
		error:    err,
	}
}

func (c *MockHttpClient) Do(req *http.Request) (*http.Response, error) {
	if c.error != nil {
		return nil, c.error
	}

	length := len(c.response)
	headers := make(http.Header)
	headers.Add("Content-Type", "application/json")

	resp := http.Response{
		StatusCode:    c.code,
		Body:          io.NopCloser(bytes.NewReader(c.response)),
		ContentLength: int64(length),
		Header:        headers,
	}
	return &resp, nil
}
