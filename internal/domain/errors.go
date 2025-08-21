package domain

import "errors"

var (
	ErrSerializationFailure = errors.New("serialization failure")
	ErrNotFound             = errors.New("not found")
	ErrConflict             = errors.New("conflict")
	ErrInvalidInput         = errors.New("invalid input")
)
