package ai

import (
	"encoding/json"
	"strings"
)

func BuildPersonDraftInput(topics json.RawMessage, source string) string {
	topics = json.RawMessage(strings.TrimSpace(string(topics)))
	if len(topics) == 0 {
		topics = json.RawMessage("[]")
	}

	return strings.Join([]string{
		"topics:",
		string(topics),
		"",
		"source_text:",
		strings.TrimSpace(source),
	}, "\n")
}
