package botmenu

import (
	"context"
	"testing"
)

func TestRootContainsPing(t *testing.T) {
	menu := New()

	root := menu.Root()
	if root == nil {
		t.Fatal("root is nil")
	}

	ping, ok := menu.Find("ping")
	if !ok {
		t.Fatal("ping item not found")
	}
	if ping.Title != "Ping" {
		t.Fatalf("ping title = %q, want %q", ping.Title, "Ping")
	}
	if !ping.IsAction() {
		t.Fatal("ping should be an action")
	}
}

func TestFindCallbackAction(t *testing.T) {
	menu := New()

	item, ok := menu.FindCallback(CallbackKey("ping"))
	if !ok {
		t.Fatal("ping callback not found")
	}

	got, err := Run(context.Background(), item)
	if err != nil {
		t.Fatalf("run ping: %v", err)
	}
	if got != "pong" {
		t.Fatalf("ping result = %q, want %q", got, "pong")
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
