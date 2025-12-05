package deploy

import "errors"

var (
	ErrInvalidConfig    = errors.New("invalid config")
	ErrInvalidFlag      = errors.New("invalid flag")
	ErrInvalidArgument  = errors.New("invalid argument")
	ErrResourceNotFound = errors.New("resource not found")
)
