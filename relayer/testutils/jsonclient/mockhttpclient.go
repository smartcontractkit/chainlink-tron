package jsonclient

import (
	"bytes"
	"io"
	"net/http"
)

type MockJsonClient struct {
	code     int
	response []byte
	error    error
}

func NewMockJsonClient(code int, response []byte, err error) *MockJsonClient {
	return &MockJsonClient{
		code:     code,
		response: response,
		error:    err,
	}
}

func (c *MockJsonClient) Do(req *http.Request) (*http.Response, error) {
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
