package ai

import (
	_ "embed"
	"strings"
)

//go:embed prompts/general_draft_system.txt
var generalDraftSystemPrompt string

//go:embed prompts/person_draft_system.txt
var personDraftSystemPrompt string

func DefaultPromptTemplates() []PromptTemplate {
	return []PromptTemplate{
		{
			Task:   TaskGeneralDraft,
			System: strings.TrimSpace(generalDraftSystemPrompt),
		},
		{
			Task:           TaskPersonDraft,
			System:         strings.TrimSpace(personDraftSystemPrompt),
			ResponseFormat: &ResponseFormat{Type: "json_object"},
		},
	}
}
