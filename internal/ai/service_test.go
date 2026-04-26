package ai

import (
	"context"
	"errors"
	"testing"
)

func TestServiceGenerateUsesDefaultModelAndPrompt(t *testing.T) {
	transport := &fakeTransport{
		response: Response{Text: "draft"},
	}
	service := NewService(
		true,
		NewRouter(true, "main", []ModelConfig{{
			ID:            "main",
			Provider:      "openai",
			Name:          "gpt-4.1-mini",
			MaxInputChars: 20,
		}}),
		NewTemplatePrompter(DefaultPromptTemplates()),
		transport,
	)

	resp, err := service.Generate(context.Background(), Request{
		Task:  TaskGeneralDraft,
		Input: "source text",
	})
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}

	if resp.Text != "draft" {
		t.Fatalf("expected response text draft, got %q", resp.Text)
	}
	if transport.model.ID != "main" {
		t.Fatalf("expected default model main, got %q", transport.model.ID)
	}
	if len(transport.prompt.Messages) != 2 {
		t.Fatalf("expected 2 prompt messages, got %d", len(transport.prompt.Messages))
	}
	if transport.prompt.Messages[0].Role != "system" || transport.prompt.Messages[1].Role != "user" {
		t.Fatalf("unexpected prompt roles: %#v", transport.prompt.Messages)
	}
}

func TestServiceGenerateReturnsDisabledError(t *testing.T) {
	service := NewService(false, NewRouter(false, "", nil), NewTemplatePrompter(DefaultPromptTemplates()), &fakeTransport{})

	_, err := service.Generate(context.Background(), Request{Task: TaskGeneralDraft, Input: "text"})
	if !errors.Is(err, ErrDisabled) {
		t.Fatalf("expected ErrDisabled, got %v", err)
	}
}

func TestServiceGenerateReturnsNotConfiguredError(t *testing.T) {
	service := NewService(true, nil, nil, nil)

	_, err := service.Generate(context.Background(), Request{Task: TaskGeneralDraft, Input: "text"})
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
}

func TestServiceGenerateRejectsUnknownModel(t *testing.T) {
	service := NewService(true, NewRouter(true, "main", []ModelConfig{{ID: "main"}}), NewTemplatePrompter(DefaultPromptTemplates()), &fakeTransport{})

	_, err := service.Generate(context.Background(), Request{Task: TaskGeneralDraft, Input: "text", ModelID: "missing"})
	if !errors.Is(err, ErrUnknownModel) {
		t.Fatalf("expected ErrUnknownModel, got %v", err)
	}
}

func TestServiceGenerateRejectsTooLongInput(t *testing.T) {
	service := NewService(
		true,
		NewRouter(true, "main", []ModelConfig{{ID: "main", MaxInputChars: 3}}),
		NewTemplatePrompter(DefaultPromptTemplates()),
		&fakeTransport{},
	)

	_, err := service.Generate(context.Background(), Request{Task: TaskGeneralDraft, Input: "абвг"})
	if !errors.Is(err, ErrInputTooLong) {
		t.Fatalf("expected ErrInputTooLong, got %v", err)
	}
}

func TestTemplatePrompterRejectsUnknownTask(t *testing.T) {
	_, err := NewTemplatePrompter(DefaultPromptTemplates()).Build(Task("unknown"), "text")
	if !errors.Is(err, ErrUnknownTask) {
		t.Fatalf("expected ErrUnknownTask, got %v", err)
	}
}

func TestTemplatePrompterRejectsEmptyInput(t *testing.T) {
	_, err := NewTemplatePrompter(DefaultPromptTemplates()).Build(TaskGeneralDraft, " \n\t ")
	if !errors.Is(err, ErrEmptyInput) {
		t.Fatalf("expected ErrEmptyInput, got %v", err)
	}
}

func TestTemplatePrompterUsesConfiguredTemplate(t *testing.T) {
	prompt, err := NewTemplatePrompter([]PromptTemplate{
		{Task: TaskGeneralDraft, System: "custom system"},
	}).Build(TaskGeneralDraft, " user text ")
	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}

	if prompt.Messages[0].Content != "custom system" {
		t.Fatalf("expected custom system prompt, got %q", prompt.Messages[0].Content)
	}
	if prompt.Messages[1].Content != "user text" {
		t.Fatalf("expected trimmed user text, got %q", prompt.Messages[1].Content)
	}
}

type fakeTransport struct {
	model    ModelConfig
	prompt   Prompt
	response Response
	err      error
}

func (t *fakeTransport) Complete(ctx context.Context, model ModelConfig, prompt Prompt) (Response, error) {
	t.model = model
	t.prompt = prompt
	return t.response, t.err
}
