package events

import (
	"context"
	"fmt"
	"log/slog"

	"proletarka_transport/internal/channels"
	"proletarka_transport/internal/domain"
)

type Dispatcher struct {
	logger    *slog.Logger
	telegram  channels.Channel
	email     channels.Channel
}

func NewDispatcher(logger *slog.Logger, telegram channels.Channel, email channels.Channel) *Dispatcher {
	return &Dispatcher{
		logger:   logger,
		telegram: telegram,
		email:    email,
	}
}

func (d *Dispatcher) Dispatch(ctx context.Context, event domain.Event) error {
	switch event.Event {
	case "comment.created":
		return d.handleCommentCreated(ctx, event)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedEvent, event.Event)
	}
}
