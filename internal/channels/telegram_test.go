package channels

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/go-telegram/bot/models"

	"proletarka_transport/internal/ai"
	"proletarka_transport/internal/backend"
)

func TestIsPlainText(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{name: "plain text", text: "что-то непонятное", want: true},
		{name: "trimmed plain text", text: "  что-то непонятное  ", want: true},
		{name: "command", text: "/unknown", want: false},
		{name: "trimmed command", text: "  /unknown  ", want: false},
		{name: "empty", text: "   ", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPlainText(tt.text); got != tt.want {
				t.Fatalf("isPlainText(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestIsCommandMessage(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{name: "command", text: "/unknown", want: true},
		{name: "trimmed command", text: "  /unknown  ", want: true},
		{name: "plain text", text: "что-то непонятное", want: false},
		{name: "empty", text: "   ", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update := messageUpdate(tt.text)
			if got := isCommandMessage(update); got != tt.want {
				t.Fatalf("isCommandMessage(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestAddPersonHandlerReturnsConfiguredAPIMessageWhenProviderNil(t *testing.T) {
	got, err := addPersonHandler(nil)(context.Background())
	if err != nil {
		t.Fatalf("addPersonHandler() returned error: %v", err)
	}
	if !strings.Contains(got, "API основного backend не настроен") {
		t.Fatalf("message = %q, want API disabled explanation", got)
	}
}

func TestAddPersonHandlerFormatsProviderTopics(t *testing.T) {
	handler := addPersonHandler(fakeImportTopicsProvider{
		topics: []backend.ImportTopic{
			{Code: "war", Title: "Война"},
		},
	})

	got, err := handler(context.Background())
	if err != nil {
		t.Fatalf("addPersonHandler() returned error: %v", err)
	}
	if !strings.Contains(got, "- Война (war)") {
		t.Fatalf("message = %q, want formatted topic", got)
	}
}

func TestAddPersonHandlerHidesProviderError(t *testing.T) {
	handler := addPersonHandler(fakeImportTopicsProvider{
		err: fmt.Errorf("backend secret raw error"),
	})

	got, err := handler(context.Background())
	if err != nil {
		t.Fatalf("addPersonHandler() returned error: %v", err)
	}
	if got != importTopicsUnavailableMessage {
		t.Fatalf("message = %q, want %q", got, importTopicsUnavailableMessage)
	}
	if strings.Contains(got, "backend secret raw error") {
		t.Fatalf("message exposes raw error: %q", got)
	}
}

func TestStartAddPersonSetsPendingState(t *testing.T) {
	channel := &TelegramChannel{}

	got := channel.startAddPerson(123)

	if got != addPersonPromptMessage {
		t.Fatalf("startAddPerson() = %q, want prompt message", got)
	}
	if channel.pendingAction(123) != waitingPersonText {
		t.Fatalf("pending action = %q, want %q", channel.pendingAction(123), waitingPersonText)
	}
}

func TestHandlePendingPersonTextGeneratesDraftAndClearsState(t *testing.T) {
	generator := &fakePersonDraftGenerator{
		response: ai.Response{Text: `{"person":{"name":"Иван Иванов","shortBio":"Токарь завода","birthYear":1900,"deathYear":null,"yearsLabel":null},"events":[{"text":"Работал на заводе","yearFrom":1942,"yearTo":null,"yearsLabel":null,"topicCodes":["war"]}],"warnings":[]}`},
	}
	channel := &TelegramChannel{
		importTopicsProvider: fakeImportTopicsProvider{
			topics: []backend.ImportTopic{
				{Code: "war", Title: "Война"},
			},
		},
		personDraftGenerator: generator,
	}
	channel.setPendingAction(123, waitingPersonText)

	got := channel.handlePendingPersonText(context.Background(), 123, " Иван Иванов, в 1942 работал на заводе. ", nil)

	if !strings.Contains(got, "Черновик для проверки") {
		t.Fatalf("result = %q, want draft title", got)
	}
	for _, want := range []string{"Человек: Иван Иванов", "Годы жизни: 1900-", "Краткое описание:", "Токарь завода", "- 1942: Работал на заводе [темы: Война]"} {
		if !strings.Contains(got, want) {
			t.Fatalf("result = %q, want %q", got, want)
		}
	}
	if generator.request.Task != ai.TaskPersonDraft {
		t.Fatalf("task = %q, want %q", generator.request.Task, ai.TaskPersonDraft)
	}
	if generator.request.ModelID != "" {
		t.Fatalf("model id = %q, want default empty model", generator.request.ModelID)
	}
	for _, want := range []string{"\"code\":\"war\"", "source_text:", "Иван Иванов, в 1942 работал на заводе."} {
		if !strings.Contains(generator.request.Input, want) {
			t.Fatalf("AI input does not contain %q: %q", want, generator.request.Input)
		}
	}
	if channel.pendingAction(123) != waitingPersonConfirm {
		t.Fatalf("pending action = %q, want %q", channel.pendingAction(123), waitingPersonConfirm)
	}
	if generator.calls != 1 {
		t.Fatalf("generator calls = %d, want 1", generator.calls)
	}
}

func TestHandlePendingPersonTextSetsAnalysisStateDuringGeneration(t *testing.T) {
	channel := &TelegramChannel{
		importTopicsProvider: fakeImportTopicsProvider{
			topics: []backend.ImportTopic{{Code: "war", Title: "Война"}},
		},
	}
	channel.personDraftGenerator = &fakePersonDraftGenerator{
		response: ai.Response{Text: `{"person":{"name":"Иван Иванов","shortBio":null,"birthYear":null,"deathYear":null,"yearsLabel":null},"events":[],"warnings":[]}`},
		onGenerate: func() {
			if channel.pendingAction(123) != waitingPersonAnalysis {
				t.Fatalf("pending action during generation = %q, want %q", channel.pendingAction(123), waitingPersonAnalysis)
			}
		},
	}
	channel.setPendingAction(123, waitingPersonText)

	_ = channel.handlePendingPersonText(context.Background(), 123, "Иван Иванов", nil)
}

func TestHandlePendingPersonTextDoesNotStartSecondAnalysis(t *testing.T) {
	generator := &fakePersonDraftGenerator{
		response: ai.Response{Text: `{"person":{"name":"Иван Иванов","shortBio":null,"birthYear":null,"deathYear":null,"yearsLabel":null},"events":[],"warnings":[]}`},
	}
	channel := &TelegramChannel{
		importTopicsProvider: fakeImportTopicsProvider{
			topics: []backend.ImportTopic{{Code: "war", Title: "Война"}},
		},
		personDraftGenerator: generator,
	}
	channel.setPendingAction(123, waitingPersonAnalysis)

	got := channel.handlePendingPersonText(context.Background(), 123, "ещё текст", nil)

	if got != personDraftInProgressMessage {
		t.Fatalf("result = %q, want %q", got, personDraftInProgressMessage)
	}
	if generator.calls != 0 {
		t.Fatalf("generator calls = %d, want 0", generator.calls)
	}
	if channel.pendingAction(123) != waitingPersonAnalysis {
		t.Fatalf("pending action = %q, want %q", channel.pendingAction(123), waitingPersonAnalysis)
	}
}

func TestHandlePendingPersonTextRejectsInvalidAIJSON(t *testing.T) {
	channel := &TelegramChannel{
		importTopicsProvider: fakeImportTopicsProvider{
			topics: []backend.ImportTopic{{Code: "war", Title: "Война"}},
		},
		personDraftGenerator: &fakePersonDraftGenerator{
			response: ai.Response{Text: "думаю, это Иван"},
		},
	}
	channel.setPendingAction(123, waitingPersonText)

	got := channel.handlePendingPersonText(context.Background(), 123, "Иван Иванов", nil)

	if got != personDraftUnavailableMessage {
		t.Fatalf("result = %q, want %q", got, personDraftUnavailableMessage)
	}
	if channel.pendingAction(123) != "" {
		t.Fatalf("pending action was not cleared: %q", channel.pendingAction(123))
	}
}

func TestConfirmPersonDraftClearsStateAndReturnsPlaceholder(t *testing.T) {
	channel := &TelegramChannel{}
	channel.setPendingAction(123, waitingPersonConfirm)

	got := channel.confirmPersonDraft(123)

	if got != personDraftConfirmPlaceholderMessage {
		t.Fatalf("confirmPersonDraft() = %q, want %q", got, personDraftConfirmPlaceholderMessage)
	}
	if channel.pendingAction(123) != "" {
		t.Fatalf("pending action was not cleared: %q", channel.pendingAction(123))
	}
}

func TestCancelPersonDraftClearsState(t *testing.T) {
	channel := &TelegramChannel{}
	channel.setPendingAction(123, waitingPersonConfirm)

	got := channel.cancelPersonDraft(123)

	if got != personDraftCancelMessage {
		t.Fatalf("cancelPersonDraft() = %q, want %q", got, personDraftCancelMessage)
	}
	if channel.pendingAction(123) != "" {
		t.Fatalf("pending action was not cleared: %q", channel.pendingAction(123))
	}
}

func TestPersonDraftActionsKeyboard(t *testing.T) {
	channel := &TelegramChannel{}

	keyboard := channel.personDraftActionsKeyboard()
	if keyboard == nil || len(keyboard.InlineKeyboard) != 1 || len(keyboard.InlineKeyboard[0]) != 2 {
		t.Fatalf("unexpected keyboard: %#v", keyboard)
	}
	if keyboard.InlineKeyboard[0][0].CallbackData != personDraftConfirmCallback {
		t.Fatalf("confirm callback = %q, want %q", keyboard.InlineKeyboard[0][0].CallbackData, personDraftConfirmCallback)
	}
	if keyboard.InlineKeyboard[0][1].CallbackData != personDraftCancelCallback {
		t.Fatalf("cancel callback = %q, want %q", keyboard.InlineKeyboard[0][1].CallbackData, personDraftCancelCallback)
	}
}

func TestHandlePendingPersonTextAsksAgainForEmptyInput(t *testing.T) {
	channel := &TelegramChannel{}
	channel.setPendingAction(123, waitingPersonText)

	got := channel.handlePendingPersonText(context.Background(), 123, "   ", nil)

	if got != addPersonPromptMessage {
		t.Fatalf("result = %q, want prompt message", got)
	}
	if channel.pendingAction(123) != waitingPersonText {
		t.Fatalf("pending action = %q, want to keep waiting state", channel.pendingAction(123))
	}
}

func TestHandlePendingPersonTextClearsStateWhenAINotConfigured(t *testing.T) {
	channel := &TelegramChannel{
		importTopicsProvider: fakeImportTopicsProvider{
			topics: []backend.ImportTopic{{Code: "war", Title: "Война"}},
		},
	}
	channel.setPendingAction(123, waitingPersonText)

	got := channel.handlePendingPersonText(context.Background(), 123, "Иван Иванов", nil)

	if !strings.Contains(got, "AI-разбор не настроен") {
		t.Fatalf("result = %q, want AI disabled explanation", got)
	}
	if channel.pendingAction(123) != "" {
		t.Fatalf("pending action was not cleared: %q", channel.pendingAction(123))
	}
}

func messageUpdate(text string) *models.Update {
	return &models.Update{
		Message: &models.Message{
			Text: text,
		},
	}
}

type fakeImportTopicsProvider struct {
	topics []backend.ImportTopic
	err    error
}

func (p fakeImportTopicsProvider) FetchImportTopics(ctx context.Context) ([]backend.ImportTopic, error) {
	return p.topics, p.err
}

type fakePersonDraftGenerator struct {
	request    ai.Request
	response   ai.Response
	err        error
	calls      int
	onGenerate func()
}

func (g *fakePersonDraftGenerator) Generate(ctx context.Context, req ai.Request) (ai.Response, error) {
	g.request = req
	g.calls++
	if g.onGenerate != nil {
		g.onGenerate()
	}
	return g.response, g.err
}
