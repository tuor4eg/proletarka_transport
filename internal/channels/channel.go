package channels

import "context"

type Message struct {
	Subject string
	Text    string
}

type Channel interface {
	Name() string
	Send(ctx context.Context, message Message) error
}
