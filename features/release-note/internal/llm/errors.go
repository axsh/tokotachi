package llm

import "errors"

// ErrNotImplemented is returned when a provider is not yet implemented.
var ErrNotImplemented = errors.New("provider not implemented")

// ErrUnknownProvider is returned when an unknown provider name is specified.
var ErrUnknownProvider = errors.New("unknown provider")
