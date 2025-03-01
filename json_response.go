// Copyright (c) 2018-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the LICENSE.md file
// distributed with the sources of this project regarding your rights to use or distribute this
// software.

package jsonresp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Error describes an error condition.
type Error struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%v (%v %v)", e.Message, e.Code, http.StatusText(e.Code))
	}
	return fmt.Sprintf("%v %v", e.Code, http.StatusText(e.Code))
}

// Is compares e against target. If target is an Error and matches the non-zero fields of e, true
// is returned.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return ((e.Code == t.Code) || t.Code == 0) &&
		((e.Message == t.Message) || t.Message == "")
}

// PageDetails specifies paging information.
type PageDetails struct {
	Prev      string `json:"prev,omitempty"`
	Next      string `json:"next,omitempty"`
	TotalSize int    `json:"totalSize,omitempty"`
}

// Response is the top level container of all of our REST API responses.
type Response struct {
	Data  interface{}  `json:"data,omitempty"`
	Page  *PageDetails `json:"page,omitempty"`
	Error *Error       `json:"error,omitempty"`
}

func encodeResponse(w http.ResponseWriter, jr Response, code int) error {
	// We _could_ encode the JSON directly to the response, but in so doing, the response code is
	// written out the first time Write() is called under the hood. This makes it difficult to
	// return an appropriate HTTP code when JSON encoding fails, so we use an intermediate buffer
	// in order to preserve our ability to set the correct HTTP code.
	b, err := json.Marshal(jr)
	if err != nil {
		return fmt.Errorf("jsonresp: failed to encode response: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if _, err := w.Write(b); err != nil {
		return fmt.Errorf("jsonresp: failed to write response: %v", err)
	}
	return nil
}

// WriteError writes a status code and JSON response containing the supplied error message and
// status code to w.
func WriteError(w http.ResponseWriter, message string, code int) error {
	jr := Response{
		Error: &Error{
			Code:    code,
			Message: message,
		},
	}
	return encodeResponse(w, jr, code)
}

// WriteResponsePage writes a status code and JSON response containing data and pd to w.
func WriteResponsePage(w http.ResponseWriter, data interface{}, pd *PageDetails, code int) error {
	jr := Response{
		Data: data,
		Page: pd,
	}
	return encodeResponse(w, jr, code)
}

// WriteResponse writes a status code and JSON response containing data to w.
func WriteResponse(w http.ResponseWriter, data interface{}, code int) error {
	return WriteResponsePage(w, data, nil, code)
}

// ReadResponsePage reads a paged JSON response, and unmarshals the supplied data.
func ReadResponsePage(r io.Reader, v interface{}) (pd *PageDetails, err error) {
	var u struct {
		Data  json.RawMessage `json:"data"`
		Page  *PageDetails    `json:"page"`
		Error *Error          `json:"error"`
	}
	if err := json.NewDecoder(r).Decode(&u); err != nil {
		return nil, fmt.Errorf("jsonresp: failed to read response: %v", err)
	}
	if u.Error != nil {
		return nil, u.Error
	}
	if v != nil {
		if err := json.Unmarshal(u.Data, v); err != nil {
			return nil, fmt.Errorf("jsonresp: failed to unmarshal response: %v", err)
		}
	}
	return u.Page, nil
}

// ReadResponse reads a JSON response, and unmarshals the supplied data.
func ReadResponse(r io.Reader, v interface{}) error {
	_, err := ReadResponsePage(r, v)
	return err
}

// ReadError attempts to unmarshal JSON-encoded error details from the supplied reader. It returns
// nil if an error could not be parsed from the response, or if the parsed error was nil.
func ReadError(r io.Reader) error {
	var u struct {
		Error *Error `json:"error"`
	}
	if err := json.NewDecoder(r).Decode(&u); err != nil {
		return nil
	}
	if u.Error == nil {
		return nil
	}
	return u.Error
}
