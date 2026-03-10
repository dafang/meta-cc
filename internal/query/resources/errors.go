package resources

import "errors"

// Sentinel errors for consistent error handling by callers.
var (
	ErrSessionLoad    = errors.New("query: session load failed")
	ErrFilterInvalid  = errors.New("query: invalid filter")
	ErrInvalidPattern = errors.New("query: invalid regex pattern")
)
