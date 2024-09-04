package solidityclient

import (
	"bytes"
	"io"
	"net/http"
)

type MockSolidityClient struct {
	code     int
	response string
	error    error
}

func NewMockSolidityClient(code int, response string, err error) *MockSolidityClient {
	return &MockSolidityClient{
		code:     code,
		response: response,
		error:    err,
	}
}

func (c *MockSolidityClient) Do(req *http.Request) (*http.Response, error) {
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
