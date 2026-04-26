package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPTransportCompleteSendsChatCompletionRequest(t *testing.T) {
	var received chatCompletionRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("expected /chat/completions, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}

		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer server.Close()

	transport := NewHTTPTransport(server.Client())
	resp, err := transport.Complete(context.Background(), ModelConfig{
		ID:       "main",
		Provider: "openai",
		Name:     "gpt-4.1-mini",
		APIKey:   "secret",
		BaseURL:  server.URL + "/",
	}, Prompt{Messages: []Message{{Role: "user", Content: "text"}}})
	if err != nil {
		t.Fatalf("Complete() returned error: %v", err)
	}

	if received.Model != "gpt-4.1-mini" {
		t.Fatalf("expected model gpt-4.1-mini, got %s", received.Model)
	}
	if len(received.Messages) != 1 || received.Messages[0].Content != "text" {
		t.Fatalf("unexpected messages: %#v", received.Messages)
	}
	if resp.Text != "ok" {
		t.Fatalf("expected response text ok, got %q", resp.Text)
	}
}

func TestHTTPTransportCompleteReturnsStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	defer server.Close()

	transport := NewHTTPTransport(server.Client())
	_, err := transport.Complete(context.Background(), ModelConfig{
		ID:       "main",
		Provider: "openai",
		Name:     "gpt-4.1-mini",
		APIKey:   "secret",
		BaseURL:  server.URL,
	}, Prompt{Messages: []Message{{Role: "user", Content: "text"}}})
	if err == nil {
		t.Fatal("expected status error")
	}
	if !strings.Contains(err.Error(), "status 502") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHTTPTransportRequiresBaseURL(t *testing.T) {
	transport := NewHTTPTransport(http.DefaultClient)
	_, err := transport.Complete(context.Background(), ModelConfig{
		ID:       "local",
		Provider: "custom",
		Name:     "model",
		APIKey:   "secret",
	}, Prompt{})
	if err == nil {
		t.Fatal("expected missing base url error")
	}
	if !strings.Contains(err.Error(), "BASE_URL is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChatCompletionsURLAcceptsVersionedBaseURL(t *testing.T) {
	got, err := chatCompletionsURL(ModelConfig{
		ID:       "main",
		Provider: "minimax",
		BaseURL:  "https://api.minimax.io/v1/",
	})
	if err != nil {
		t.Fatalf("chatCompletionsURL() returned error: %v", err)
	}

	want := "https://api.minimax.io/v1/chat/completions"
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}
