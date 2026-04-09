package domain

import (
	"fmt"
	"strings"
	"time"
)

type Event struct {
	Event      string         `json:"event"`
	OccurredAt string         `json:"occurredAt"`
	Severity   string         `json:"severity"`
	Resource   EventResource  `json:"resource"`
	Payload    map[string]any `json:"payload"`
}

type EventResource struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

func (e Event) Validate() error {
	if strings.TrimSpace(e.Event) == "" {
		return fmt.Errorf("event is required")
	}

	if strings.TrimSpace(e.OccurredAt) == "" {
		return fmt.Errorf("occurredAt is required")
	}

	if _, err := time.Parse(time.RFC3339, e.OccurredAt); err != nil {
		return fmt.Errorf("occurredAt must be a valid RFC3339 timestamp")
	}

	switch e.Severity {
	case "low", "normal", "high":
	default:
		return fmt.Errorf("severity must be one of: low, normal, high")
	}

	if strings.TrimSpace(e.Resource.Kind) == "" {
		return fmt.Errorf("resource.kind is required")
	}

	if strings.TrimSpace(e.Resource.ID) == "" {
		return fmt.Errorf("resource.id is required")
	}

	if e.Payload == nil {
		return fmt.Errorf("payload is required")
	}

	return nil
}
