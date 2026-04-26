package ai

import (
	"fmt"
	"strings"
)

type PromptTemplate struct {
	Task           Task
	System         string
	ResponseFormat *ResponseFormat
}

type TemplatePrompter struct {
	templates map[Task]PromptTemplate
}

func NewTemplatePrompter(templates []PromptTemplate) TemplatePrompter {
	byTask := make(map[Task]PromptTemplate, len(templates))
	for _, template := range templates {
		byTask[template.Task] = template
	}

	return TemplatePrompter{templates: byTask}
}

func (p TemplatePrompter) Build(task Task, input string) (Prompt, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return Prompt{}, ErrEmptyInput
	}

	template, ok := p.templates[task]
	if !ok {
		return Prompt{}, fmt.Errorf("%w: %s", ErrUnknownTask, task)
	}

	return Prompt{
		Messages: []Message{
			{
				Role:    "system",
				Content: template.System,
			},
			{
				Role:    "user",
				Content: input,
			},
		},
		ResponseFormat: template.ResponseFormat,
	}, nil
}
