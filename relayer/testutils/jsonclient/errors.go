package jsonclient

import (
	"encoding/json"
	"fmt"
)

type MarshalError struct {
	message string
}

func (e *MarshalError) Error() string {
	return fmt.Sprintf("payload marshal failed: %v", e.message)
}

func NewMarshalError(err error) *MarshalError {
	return &MarshalError{err.Error()}
}

type UnmarshalError struct {
	message string
}

func (e *UnmarshalError) Error() string {
	return fmt.Sprintf("response unmarshal failed: %v", e.message)
}

func NewUnmarshalError(err error) *UnmarshalError {
	return &UnmarshalError{err.Error()}
}

type RequestCreationError struct {
	url     string
	message string
}

func (e *RequestCreationError) Error() string {
	return fmt.Sprintf("request creation failed (url: %s): %v", e.url, e.message)
}

func NewRequestCreationError(endpoint string, err error) *RequestCreationError {
	return &RequestCreationError{endpoint, err.Error()}
}

type RequestError struct {
	url     string
	message string
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("http request failed (url: %s): %v", e.url, e.message)
}

func NewRequestError(endpoint string, err error) *RequestError {
	return &RequestError{endpoint, err.Error()}
}

type ResponseBodyError struct {
	url     string
	message string
}

func (e *ResponseBodyError) Error() string {
	return fmt.Sprintf("reading response body failed (url: %s): %v", e.url, e.message)
}

func NewResponseBodyError(endpoint string, err error) *ResponseBodyError {
	return &ResponseBodyError{endpoint, err.Error()}
}

type StatusCodeError struct {
	url           string
	errorresponse string
	code          int
}

func (e *StatusCodeError) Error() string {
	return fmt.Sprintf("invalid http response code (url: %s) http %d: %v", e.url, e.code, e.errorresponse)
}

func NewStatusCodeError(endpoint string, body []byte, status int) *StatusCodeError {
	var response map[string]interface{}
	var errstr string

	if unmarshalError := json.Unmarshal(body, &response); unmarshalError != nil {
		umerr := NewUnmarshalError(unmarshalError)
		errstr = umerr.Error()
	} else {
		if errResponse, found := response["Error"].(string); found {
			errstr = errResponse
		}
	}

	return &StatusCodeError{endpoint, errstr, status}
}
