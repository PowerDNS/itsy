package itsy

import (
	"errors"

	"github.com/nats-io/nats.go/micro"
)

type Request struct {
	mr   micro.Request
	opts []micro.RespondOpt
}

// Respond sends the response for the request.
// Additional headers can be passed using [WithHeaders] option.
func (r Request) Respond(msg []byte, opts ...micro.RespondOpt) error {
	opts = append(opts, r.opts...)
	return r.mr.Respond(msg, opts...)
}

// RespondJSON marshals the given response value and responds to the request.
// Additional headers can be passed using [WithHeaders] option.
func (r Request) RespondJSON(data any, opts ...micro.RespondOpt) error {
	opts = append(opts, r.opts...)
	return r.mr.RespondJSON(data, opts...)
}

// Error prepares and publishes error response from a handler.
// A response error should be set containing an error code and description.
// Optionally, data can be set as response payload.
func (r Request) Error(code, description string, data []byte, opts ...micro.RespondOpt) error {
	opts = append(opts, r.opts...)
	return r.mr.Error(code, description, data, opts...)
}

// Data returns request data.
func (r Request) Data() []byte {
	return r.mr.Data()
}

// Headers returns request headers.
func (r Request) Headers() micro.Headers {
	return r.mr.Headers()
}

// Subject returns underlying NATS message subject.
func (r Request) Subject() string {
	return r.mr.Subject()
}

// Verify that this implements the micro.Request interface
var _ micro.Request = &Request{}

// Extra extensions added by us

// Err provides an easy way to return a Go error as error
// Use ErrorResponse to add a custom code. Wrap can create one for you.
func (r Request) Err(err error, opts ...micro.RespondOpt) error {
	code := "ERR"
	var er ErrorResponse
	if errors.As(err, &er) {
		code = er.Code
	}
	opts = append(opts, r.opts...)
	return r.mr.Error(code, err.Error(), nil, opts...)
}

// ErrorResponse adds a description code to the error
type ErrorResponse struct {
	Code string // Code to be returned with the NATS error
	Err  error  // Actual error
}

func (er ErrorResponse) Error() string {
	return er.Err.Error()
}

func Wrap(err error, code string) ErrorResponse {
	return ErrorResponse{
		Code: code,
		Err:  err,
	}
}
