package jsonclient

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
)

type JsonHttpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type MockJsonClient struct {
	code     int
	response string
	error    *url.Error
}

func NewMockJsonClient(code int, response string, err *url.Error) *MockJsonClient {
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
	bodybytes := []byte(c.response)
	length := len(bodybytes)
	headers := make(http.Header)
	headers.Add("Content-Type", "application/json")

	resp := http.Response{
		StatusCode:    c.code,
		Body:          io.NopCloser(bytes.NewReader(bodybytes)),
		ContentLength: int64(length),
		Header:        headers,
	}
	return &resp, nil
}
