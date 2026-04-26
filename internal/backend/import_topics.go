package backend

import (
	"context"
	"fmt"
	"strings"
)

type ImportTopic struct {
	Code       string        `json:"code"`
	Title      string        `json:"title"`
	ParentCode *string       `json:"parentCode"`
	IsSystem   bool          `json:"isSystem"`
	Children   []ImportTopic `json:"children"`
}

type importTopicsResponse struct {
	OK     bool          `json:"ok"`
	Topics []ImportTopic `json:"topics"`
}

func (c *Client) FetchImportTopics(ctx context.Context) ([]ImportTopic, error) {
	var response importTopicsResponse
	if err := c.Get(ctx, "/api/import/topics", &response); err != nil {
		return nil, fmt.Errorf("fetch import topics: %w", err)
	}
	if !response.OK {
		return nil, fmt.Errorf("fetch import topics: backend returned ok=false")
	}

	return response.Topics, nil
}

func FormatImportTopics(topics []ImportTopic) string {
	if len(topics) == 0 {
		return "Темы для импорта пока не настроены."
	}

	var builder strings.Builder
	builder.WriteString("Доступные темы для добавления человека:\n\n")
	writeImportTopics(&builder, topics, 0)

	result := strings.TrimRight(builder.String(), "\n")
	if result == "Доступные темы для добавления человека:" {
		return "Темы для импорта пока не настроены."
	}

	return result
}

func writeImportTopics(builder *strings.Builder, topics []ImportTopic, depth int) {
	indent := strings.Repeat("  ", depth)
	for _, topic := range topics {
		title := strings.TrimSpace(topic.Title)
		if title == "" {
			title = strings.TrimSpace(topic.Code)
		}
		if title == "" {
			continue
		}

		builder.WriteString(indent)
		builder.WriteString("- ")
		builder.WriteString(title)
		if code := strings.TrimSpace(topic.Code); code != "" {
			builder.WriteString(" (")
			builder.WriteString(code)
			builder.WriteString(")")
		}
		builder.WriteByte('\n')

		if len(topic.Children) > 0 {
			writeImportTopics(builder, topic.Children, depth+1)
		}
	}
}
