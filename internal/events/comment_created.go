package events

import (
	"context"
	"fmt"
	"strings"

	"proletarka_transport/internal/channels"
	"proletarka_transport/internal/domain"
)

type commentCreatedPayload struct {
	CommentText       string
	CommentAuthorName string
	TargetTitle       string
	TargetType        string
	PublicURL         string
	AdminURL          string
}

func (d *Dispatcher) handleCommentCreated(ctx context.Context, event domain.Event) error {
	payload, err := parseCommentCreatedPayload(event.Payload)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidEvent, err)
	}

	message := buildCommentCreatedMessage(event, payload)
	var attempts []string

	if d.telegram != nil {
		d.logger.Info("attempting delivery", "event", event.Event, "channel", d.telegram.Name(), "resource_kind", event.Resource.Kind, "resource_id", event.Resource.ID)
		if err := d.telegram.Send(ctx, message); err == nil {
			d.logger.Info("delivery succeeded", "event", event.Event, "channel", d.telegram.Name(), "resource_kind", event.Resource.Kind, "resource_id", event.Resource.ID)
			return nil
		} else {
			d.logger.Warn("delivery failed", "event", event.Event, "channel", d.telegram.Name(), "resource_kind", event.Resource.Kind, "resource_id", event.Resource.ID, "error", err.Error())
			attempts = append(attempts, fmt.Sprintf("%s: %v", d.telegram.Name(), err))
		}
	}

	if d.email != nil {
		d.logger.Info("attempting delivery", "event", event.Event, "channel", d.email.Name(), "resource_kind", event.Resource.Kind, "resource_id", event.Resource.ID)
		if err := d.email.Send(ctx, message); err == nil {
			d.logger.Info("delivery succeeded", "event", event.Event, "channel", d.email.Name(), "resource_kind", event.Resource.Kind, "resource_id", event.Resource.ID)
			return nil
		} else {
			d.logger.Warn("delivery failed", "event", event.Event, "channel", d.email.Name(), "resource_kind", event.Resource.Kind, "resource_id", event.Resource.ID, "error", err.Error())
			attempts = append(attempts, fmt.Sprintf("%s: %v", d.email.Name(), err))
		}
	}

	return fmt.Errorf("%w for comment.created: %s", ErrDeliveryFailed, strings.Join(attempts, "; "))
}

func parseCommentCreatedPayload(payload map[string]any) (commentCreatedPayload, error) {
	comment := getMap(payload, "comment")
	target := getMap(payload, "target")
	urls := getMap(payload, "urls")

	result := commentCreatedPayload{
		CommentText:       firstNonEmpty(getString(comment, "text"), getString(payload, "commentText")),
		CommentAuthorName: firstNonEmpty(getString(comment, "authorName"), getString(comment, "author"), getString(payload, "commentAuthorName")),
		TargetTitle:       firstNonEmpty(getString(target, "title"), getString(payload, "targetTitle")),
		TargetType:        firstNonEmpty(getString(target, "type"), getString(payload, "targetType")),
		PublicURL:         firstNonEmpty(getString(urls, "public"), getString(payload, "publicUrl"), getString(payload, "publicURL")),
		AdminURL:          firstNonEmpty(getString(urls, "admin"), getString(payload, "adminUrl"), getString(payload, "adminURL")),
	}

	if result.CommentText == "" {
		return commentCreatedPayload{}, fmt.Errorf("comment.created payload must include comment text")
	}

	return result, nil
}

func buildCommentCreatedMessage(event domain.Event, payload commentCreatedPayload) channels.Message {
	subjectTarget := payload.TargetTitle
	if subjectTarget == "" {
		subjectTarget = event.Resource.Kind
	}

	lines := []string{
		"New comment created",
		fmt.Sprintf("Severity: %s", event.Severity),
		fmt.Sprintf("Resource: %s/%s", event.Resource.Kind, event.Resource.ID),
	}

	if payload.CommentAuthorName != "" {
		lines = append(lines, fmt.Sprintf("Author: %s", payload.CommentAuthorName))
	}

	if payload.TargetType != "" {
		lines = append(lines, fmt.Sprintf("Target type: %s", payload.TargetType))
	}

	if payload.TargetTitle != "" {
		lines = append(lines, fmt.Sprintf("Target title: %s", payload.TargetTitle))
	}

	lines = append(lines, "", "Comment:", payload.CommentText)

	if payload.PublicURL != "" {
		lines = append(lines, "", fmt.Sprintf("Public: %s", payload.PublicURL))
	}

	if payload.AdminURL != "" {
		lines = append(lines, fmt.Sprintf("Admin: %s", payload.AdminURL))
	}

	return channels.Message{
		Subject: fmt.Sprintf("New comment on %s", subjectTarget),
		Text:    strings.Join(lines, "\n"),
	}
}

func getMap(value map[string]any, key string) map[string]any {
	raw, ok := value[key]
	if !ok {
		return nil
	}

	result, ok := raw.(map[string]any)
	if !ok {
		return nil
	}

	return result
}

func getString(value map[string]any, key string) string {
	if value == nil {
		return ""
	}

	raw, ok := value[key]
	if !ok {
		return ""
	}

	text, ok := raw.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(text)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}

	return ""
}
