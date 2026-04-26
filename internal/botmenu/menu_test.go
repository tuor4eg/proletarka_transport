package botmenu

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRootContainsActions(t *testing.T) {
	menu := New()

	root := menu.Root()
	if root == nil {
		t.Fatal("root is nil")
	}

	addPerson, ok := menu.Find("add_person")
	if !ok {
		t.Fatal("add_person item not found")
	}
	if addPerson.Title != "Добавить человека" {
		t.Fatalf("add_person title = %q, want %q", addPerson.Title, "Добавить человека")
	}
	if !addPerson.IsAction() {
		t.Fatal("add_person should be an action")
	}

	ping, ok := menu.Find("ping")
	if !ok {
		t.Fatal("ping item not found")
	}
	if ping.Title != "Проверить связь" {
		t.Fatalf("ping title = %q, want %q", ping.Title, "Проверить связь")
	}
	if !ping.IsAction() {
		t.Fatal("ping should be an action")
	}
}

func TestAddPersonReturnsPlaceholder(t *testing.T) {
	menu := New()

	item, ok := menu.FindCallback(CallbackKey("add_person"))
	if !ok {
		t.Fatal("add_person callback not found")
	}

	got, err := Run(context.Background(), item)
	if err != nil {
		t.Fatalf("run add_person: %v", err)
	}
	for _, want := range []string{"Добавление человека пока в подготовке.", "описание человека", "черновик", "подтвердить"} {
		if !strings.Contains(got, want) {
			t.Fatalf("add_person result %q does not contain %q", got, want)
		}
	}
}

func TestAddPersonUsesConfiguredHandler(t *testing.T) {
	menu := NewWithOptions(Options{
		AddPersonHandler: func(ctx context.Context) (string, error) {
			return "Темы загружены", nil
		},
	})

	item, ok := menu.FindCallback(CallbackKey("add_person"))
	if !ok {
		t.Fatal("add_person callback not found")
	}

	got, err := Run(context.Background(), item)
	if err != nil {
		t.Fatalf("run add_person: %v", err)
	}
	if got != "Темы загружены" {
		t.Fatalf("add_person result = %q, want %q", got, "Темы загружены")
	}
}

func TestFindCallbackAction(t *testing.T) {
	menu := NewWithStartedAt(time.Now().Add(-2*time.Hour - 3*time.Minute - 4*time.Second))

	item, ok := menu.FindCallback(CallbackKey("ping"))
	if !ok {
		t.Fatal("ping callback not found")
	}

	got, err := Run(context.Background(), item)
	if err != nil {
		t.Fatalf("run ping: %v", err)
	}
	for _, want := range []string{"Proletarka transport на связи.", "Статус: работает", "Аптайм:"} {
		if !strings.Contains(got, want) {
			t.Fatalf("ping result %q does not contain %q", got, want)
		}
	}
	if !strings.Contains(got, "2 ч") {
		t.Fatalf("ping result %q does not contain expected uptime hours", got)
	}
}

func TestFindRootTitle(t *testing.T) {
	menu := New()

	addPerson, ok := menu.FindRootTitle("Добавить человека")
	if !ok {
		t.Fatal("add_person root title not found")
	}
	if addPerson.ID != "add_person" {
		t.Fatalf("root title item = %q, want %q", addPerson.ID, "add_person")
	}

	item, ok := menu.FindRootTitle("Проверить связь")
	if !ok {
		t.Fatal("ping root title not found")
	}
	if item.ID != "ping" {
		t.Fatalf("root title item = %q, want %q", item.ID, "ping")
	}
}

func TestFindSupportsNestedItems(t *testing.T) {
	menu := NewWithRoot(&Item{
		ID:    "root",
		Title: "Root",
		Children: []*Item{
			{
				ID:    "tools",
				Title: "Tools",
				Children: []*Item{
					{ID: "nested-ping", Title: "Nested Ping"},
				},
			},
		},
	})

	item, ok := menu.FindCallback(CallbackKey("nested-ping"))
	if !ok {
		t.Fatal("nested item not found by callback")
	}
	if item.Title != "Nested Ping" {
		t.Fatalf("nested title = %q, want %q", item.Title, "Nested Ping")
	}
}

func TestParentSupportsNestedItems(t *testing.T) {
	menu := NewWithRoot(&Item{
		ID:    "root",
		Title: "Root",
		Children: []*Item{
			{
				ID:    "tools",
				Title: "Tools",
				Children: []*Item{
					{ID: "nested-ping", Title: "Nested Ping"},
				},
			},
		},
	})

	parent, ok := menu.Parent("nested-ping")
	if !ok {
		t.Fatal("nested parent not found")
	}
	if parent.ID != "tools" {
		t.Fatalf("nested parent = %q, want %q", parent.ID, "tools")
	}

	if _, ok := menu.Parent("root"); ok {
		t.Fatal("root should not have parent")
	}
}
