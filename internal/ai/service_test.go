package ai

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
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
	if prompt.ResponseFormat != nil {
		t.Fatalf("expected no response format, got %#v", prompt.ResponseFormat)
	}
}

func TestTemplatePrompterBuildsPersonDraftPrompt(t *testing.T) {
	prompt, err := NewTemplatePrompter(DefaultPromptTemplates()).Build(TaskPersonDraft, "source")
	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}
	if prompt.ResponseFormat == nil {
		t.Fatal("expected person draft response format")
	}
	if prompt.ResponseFormat.Type != "json_object" {
		t.Fatalf("expected response format json_object, got %q", prompt.ResponseFormat.Type)
	}

	system := prompt.Messages[0].Content
	for _, want := range []string{
		"Верни только JSON формата",
		"topicCodes",
		"Не выдумывай факты",
		"parent topic codes",
		"Не дублируй одну и ту же информацию",
	} {
		if !strings.Contains(system, want) {
			t.Fatalf("person draft system prompt does not contain %q: %q", want, system)
		}
	}
}

func TestBuildPersonDraftInput(t *testing.T) {
	got := BuildPersonDraftInput(json.RawMessage(`[{"code":"war","title":"Война"}]`), " Иван Иванов. В 1942 работал на заводе. ")

	for _, want := range []string{
		"topics:\n[{\"code\":\"war\",\"title\":\"Война\"}]",
		"source_text:\nИван Иванов. В 1942 работал на заводе.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("person draft input does not contain %q: %q", want, got)
		}
	}
}

func TestParsePersonDraftRejectsInvalidJSON(t *testing.T) {
	_, err := ParsePersonDraft("не json")
	if err == nil {
		t.Fatal("expected invalid JSON error")
	}
}

func TestParsePersonDraftExtractsJSONFromReasoningText(t *testing.T) {
	draft, err := ParsePersonDraft(`<think>Сначала разберу текст.</think>
{"person":{"name":"Иван Иванов","shortBio":null,"birthYear":null,"deathYear":null,"yearsLabel":null},"events":[],"warnings":["Неясна дата рождения."]}`)
	if err != nil {
		t.Fatalf("ParsePersonDraft() returned error: %v", err)
	}

	if draft.Person.Name != "Иван Иванов" {
		t.Fatalf("person name = %q, want Иван Иванов", draft.Person.Name)
	}
	if len(draft.Warnings) != 1 {
		t.Fatalf("warnings len = %d, want 1", len(draft.Warnings))
	}
}

func TestFormatPersonDraft(t *testing.T) {
	shortBio := "Токарь завода"
	warning := "Неясна дата смерти."
	birthYear := 1900
	workYear := 1942

	got := FormatPersonDraft(PersonDraft{
		Person: DraftPerson{
			Name:      "Иван Иванов",
			ShortBio:  &shortBio,
			BirthYear: &birthYear,
		},
		Events: []DraftEvent{
			{
				Text:       "Работал на заводе",
				YearFrom:   &workYear,
				TopicCodes: []string{"war", "factory"},
			},
		},
		Warnings: []string{warning},
	}, map[string]string{
		"war":     "Война",
		"factory": "Завод",
	})

	for _, want := range []string{
		"Черновик для проверки",
		"Человек: Иван Иванов",
		"Годы жизни: 1900-",
		"Краткое описание:\nТокарь завода",
		"- 1942: Работал на заводе [темы: Война, Завод]",
		"Предупреждения:\n- Неясна дата смерти.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("formatted draft %q does not contain %q", got, want)
		}
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
