package events

import "errors"

var (
	ErrInvalidEvent     = errors.New("invalid event")
	ErrUnsupportedEvent = errors.New("unsupported event")
	ErrDeliveryFailed   = errors.New("delivery failed")
)
