package backend

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientFetchImportTopicsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/import/topics" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"ok":true,"topics":[{"code":"war","title":"Война","parentCode":null,"isSystem":false,"children":[{"code":"front","title":"Фронт","parentCode":"war","isSystem":false,"children":[]}]}]}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	topics, err := client.FetchImportTopics(context.Background())
	if err != nil {
		t.Fatalf("FetchImportTopics() returned error: %v", err)
	}
	if len(topics) != 1 {
		t.Fatalf("topics len = %d, want 1", len(topics))
	}
	if topics[0].Code != "war" || topics[0].Title != "Война" {
		t.Fatalf("unexpected topic: %+v", topics[0])
	}
	if len(topics[0].Children) != 1 || topics[0].Children[0].Code != "front" {
		t.Fatalf("unexpected children: %+v", topics[0].Children)
	}
}

func TestClientFetchImportTopicsReturnsErrorWhenOKFalse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"ok":false,"topics":[]}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	_, err := client.FetchImportTopics(context.Background())
	if err == nil {
		t.Fatal("expected ok=false error")
	}
	if !strings.Contains(err.Error(), "ok=false") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFormatImportTopicsNested(t *testing.T) {
	got := FormatImportTopics([]ImportTopic{
		{
			Code:  "war",
			Title: "Война",
			Children: []ImportTopic{
				{Code: "front", Title: "Фронт"},
				{Code: "homefront", Title: "Тыл"},
			},
		},
		{Code: "factory", Title: "Завод"},
	})

	for _, want := range []string{
		"Доступные темы для добавления человека:",
		"- Война (war)",
		"  - Фронт (front)",
		"  - Тыл (homefront)",
		"- Завод (factory)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("formatted topics %q does not contain %q", got, want)
		}
	}
}

func TestFormatImportTopicsReturnsEmptyMessageWhenTopicsHaveNoLabels(t *testing.T) {
	got := FormatImportTopics([]ImportTopic{{Children: []ImportTopic{{}}}})
	if got != "Темы для импорта пока не настроены." {
		t.Fatalf("formatted topics = %q, want empty message", got)
	}
}
