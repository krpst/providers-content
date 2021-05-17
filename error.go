package main

import (
	"errors"
	"fmt"
)

const (
	// ErrInternal returned when internal error occurs.
	ErrInternal = "internal"
	// ErrValidation is returned when the request failed validation.
	ErrValidation = "validation"
)

// Error represents an error within the context.
type Error struct {
	// Code is a machine-readable code.
	Code string `json:"code"`
	// Message is a human-readable message.
	Message string `json:"message"`
}

func (e Error) Error() string {
	return fmt.Sprintf("%s %s", e.Code, e.Message)
}

// ErrorCode returns the code of the error, if available.
func ErrorCode(err error) string {
	var e Error
	if errors.As(err, &e) {
		return e.Code
	}

	return ErrInternal
}

// ErrorMessage returns the human-readable message of the error, if available.
// Otherwise returns a generic error message.
func ErrorMessage(err error) string {
	var e Error
	if errors.As(err, &e) {
		return e.Message
	}

	return "An internal error has occurred."
}
