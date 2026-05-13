package providererrors

import (
	"fmt"
	"strings"
)

type Code string

const (
	CodeTimeout         Code = "timeout"
	CodeNetwork         Code = "network"
	CodeRateLimited     Code = "rate_limited"
	CodeNotFound        Code = "not_found"
	CodeInvalidResponse Code = "invalid_response"
	CodeUnauthorized    Code = "unauthorized"
	CodeForbidden       Code = "forbidden"
	CodeConfiguration   Code = "configuration"
	CodeUnsupported     Code = "unsupported"
	CodeUnavailable     Code = "unavailable"
	CodeUpstream        Code = "upstream"
	CodeUnknown         Code = "unknown"
)

type Options struct {
	Provider   string
	Operation  string
	Code       Code
	Message    string
	StatusCode int
	Retryable  bool
	Cause      error
	RequestURL string
}

type Error struct {
	Provider   string
	Operation  string
	Code       Code
	Message    string
	StatusCode int
	Retryable  bool
	RequestURL string
	cause      error
}

func New(opts Options) *Error {
	code := opts.Code
	if code == "" {
		code = CodeUnknown
	}

	message := strings.TrimSpace(opts.Message)
	if message == "" {
		message = "Provider request failed."
	}

	return &Error{
		Provider:   strings.TrimSpace(opts.Provider),
		Operation:  strings.TrimSpace(opts.Operation),
		Code:       code,
		Message:    message,
		StatusCode: opts.StatusCode,
		Retryable:  opts.Retryable,
		RequestURL: strings.TrimSpace(opts.RequestURL),
		cause:      opts.Cause,
	}
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}

	prefix := strings.TrimSpace(strings.Join([]string{
		strings.TrimSpace(e.Provider),
		strings.TrimSpace(e.Operation),
	}, " "))
	if prefix == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", prefix, e.Message)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func Configuration(provider, operation, message string) *Error {
	return New(Options{
		Provider:  provider,
		Operation: operation,
		Code:      CodeConfiguration,
		Message:   message,
	})
}

func InvalidResponse(provider, operation, message string) *Error {
	if strings.TrimSpace(message) == "" {
		message = "Provider returned an invalid response."
	}
	return New(Options{
		Provider:  provider,
		Operation: operation,
		Code:      CodeInvalidResponse,
		Message:   message,
	})
}

func Unsupported(provider, operation, message string) *Error {
	if strings.TrimSpace(message) == "" {
		message = "Provider operation is unsupported."
	}
	return New(Options{
		Provider:  provider,
		Operation: operation,
		Code:      CodeUnsupported,
		Message:   message,
	})
}

func IsRetryableCode(code Code) bool {
	switch code {
	case CodeTimeout, CodeNetwork, CodeRateLimited, CodeUpstream:
		return true
	default:
		return false
	}
}
