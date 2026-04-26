package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"proletarka_transport/internal/config"
)

func TestClientGetAddsHeadersBuildsPathAndDecodesJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/base/api/foo" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.RawQuery != "q=one" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Fatalf("unexpected Accept header: %s", r.Header.Get("Accept"))
		}
		if r.Header.Get("X-Backend-Secret") != "backend-secret" {
			t.Fatalf("unexpected auth header")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"ok":true,"name":"factory"}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+"/base")

	var out struct {
		OK   bool   `json:"ok"`
		Name string `json:"name"`
	}
	if err := client.Get(context.Background(), "api/foo?q=one", &out); err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}

	if !out.OK || out.Name != "factory" {
		t.Fatalf("unexpected decoded response: %+v", out)
	}
}

func TestClientPostAddsHeadersSendsBodyAndDecodesJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/foo" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("unexpected Content-Type header: %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Fatalf("unexpected Accept header: %s", r.Header.Get("Accept"))
		}
		if r.Header.Get("X-Backend-Secret") != "backend-secret" {
			t.Fatalf("unexpected auth header")
		}

		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body.Name != "factory" {
			t.Fatalf("unexpected request body: %+v", body)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"id":42}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	var out struct {
		ID int `json:"id"`
	}
	err := client.Post(context.Background(), "/api/foo", map[string]string{"name": "factory"}, &out)
	if err != nil {
		t.Fatalf("Post() returned error: %v", err)
	}

	if out.ID != 42 {
		t.Fatalf("unexpected decoded response: %+v", out)
	}
}

func TestClientAllowsEmptySuccessBodyWhenOutIsNil(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	if err := client.Get(context.Background(), "/api/empty", nil); err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}
}

func TestClientReturnsNon2xxErrorWithLimitedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, strings.Repeat("x", maxErrorBodyBytes+100), http.StatusBadGateway)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	err := client.Get(context.Background(), "/api/fail", nil)
	if err == nil {
		t.Fatal("expected non-2xx error")
	}
	if !strings.Contains(err.Error(), "status 502") {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(err.Error()) > maxErrorBodyBytes+128 {
		t.Fatalf("expected limited error body, got error length %d", len(err.Error()))
	}
	if strings.Contains(err.Error(), "backend-secret") {
		t.Fatalf("error must not contain secret: %v", err)
	}
}

func TestClientRejectsAbsoluteURLPath(t *testing.T) {
	client := newTestClient(t, "https://backend.example.com")

	err := client.Get(context.Background(), "https://evil.example.com/api/foo", nil)
	if err == nil {
		t.Fatal("expected error for absolute path")
	}
	if !strings.Contains(err.Error(), "backend path must be relative") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func newTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()

	client, err := NewClient(config.APIConfig{
		BaseURL:   baseURL,
		HeaderKey: "X-Backend-Secret",
		Secret:    "backend-secret",
		Enabled:   true,
	}, nil)
	if err != nil {
		t.Fatalf("NewClient() returned error: %v", err)
	}

	return client
}
