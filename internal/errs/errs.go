package errs

import "errors"

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrInvalidPIN      = errors.New("invalid pin")
)
