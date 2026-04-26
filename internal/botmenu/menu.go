package botmenu

import (
	"context"
	"fmt"
	"strings"
)

const CallbackPrefix = "menu:"

type Action func(ctx context.Context) (string, error)

type Item struct {
	ID       string
	Title    string
	Action   Action
	Children []*Item
}

func (i *Item) IsAction() bool {
	return i != nil && i.Action != nil
}

type Menu struct {
	root *Item
}

func New() *Menu {
	return &Menu{
		root: &Item{
			ID:    "root",
			Title: "Меню",
			Children: []*Item{
				{
					ID:    "ping",
					Title: "Ping",
					Action: func(ctx context.Context) (string, error) {
						return "pong", nil
					},
				},
			},
		},
	}
}

func NewWithRoot(root *Item) *Menu {
	return &Menu{root: root}
}

func (m *Menu) Root() *Item {
	if m == nil {
		return nil
	}

	return m.root
}

func (m *Menu) Find(id string) (*Item, bool) {
	if m == nil || m.root == nil || id == "" {
		return nil, false
	}

	return find(m.root, id)
}

func (m *Menu) Parent(id string) (*Item, bool) {
	if m == nil || m.root == nil || id == "" || m.root.ID == id {
		return nil, false
	}

	return parent(m.root, id)
}

func (m *Menu) FindCallback(data string) (*Item, bool) {
	id, ok := CallbackID(data)
	if !ok {
		return nil, false
	}

	return m.Find(id)
}

func CallbackKey(id string) string {
	return CallbackPrefix + id
}

func CallbackID(data string) (string, bool) {
	if !strings.HasPrefix(data, CallbackPrefix) {
		return "", false
	}

	id := strings.TrimPrefix(data, CallbackPrefix)
	if id == "" {
		return "", false
	}

	return id, true
}

func Run(ctx context.Context, item *Item) (string, error) {
	if item == nil || item.Action == nil {
		return "", fmt.Errorf("menu item has no action")
	}

	return item.Action(ctx)
}

func find(item *Item, id string) (*Item, bool) {
	if item.ID == id {
		return item, true
	}

	for _, child := range item.Children {
		if found, ok := find(child, id); ok {
			return found, true
		}
	}

	return nil, false
}

func parent(item *Item, id string) (*Item, bool) {
	for _, child := range item.Children {
		if child.ID == id {
			return item, true
		}

		if found, ok := parent(child, id); ok {
			return found, true
		}
	}

	return nil, false
}
