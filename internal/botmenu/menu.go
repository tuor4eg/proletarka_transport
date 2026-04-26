package botmenu

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const CallbackPrefix = "menu:"

type Action func(ctx context.Context) (string, error)
type AddPersonAction func(ctx context.Context) (string, error)

type Options struct {
	AddPersonHandler AddPersonAction
}

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
	root      *Item
	startedAt time.Time
}

func New() *Menu {
	return NewWithStartedAt(time.Now())
}

func NewWithOptions(options Options) *Menu {
	return newWithStartedAtAndOptions(time.Now(), options)
}

func NewWithStartedAt(startedAt time.Time) *Menu {
	return newWithStartedAtAndOptions(startedAt, Options{})
}

func newWithStartedAtAndOptions(startedAt time.Time, options Options) *Menu {
	if startedAt.IsZero() {
		startedAt = time.Now()
	}

	menu := &Menu{startedAt: startedAt}
	addPersonAction := AddPersonAction(func(ctx context.Context) (string, error) {
		return addPersonPlaceholderMessage(), nil
	})
	if options.AddPersonHandler != nil {
		addPersonAction = options.AddPersonHandler
	}

	menu.root = &Item{
		ID:    "root",
		Title: "Меню",
		Children: []*Item{
			{
				ID:     "add_person",
				Title:  "Добавить человека",
				Action: Action(addPersonAction),
			},
			{
				ID:    "ping",
				Title: "Проверить связь",
				Action: func(ctx context.Context) (string, error) {
					return statusMessage(time.Since(menu.startedAt)), nil
				},
			},
		},
	}

	return menu
}

func NewWithRoot(root *Item) *Menu {
	return &Menu{root: root, startedAt: time.Now()}
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

func (m *Menu) FindRootTitle(title string) (*Item, bool) {
	if m == nil || m.root == nil || title == "" {
		return nil, false
	}

	for _, child := range m.root.Children {
		if child.Title == title {
			return child, true
		}
	}

	return nil, false
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

func statusMessage(uptime time.Duration) string {
	return fmt.Sprintf(
		"Proletarka transport на связи.\n\nСтатус: работает\nАптайм: %s",
		formatUptime(uptime),
	)
}

func addPersonPlaceholderMessage() string {
	return "Добавление человека пока в подготовке.\n\nСкоро здесь можно будет отправить описание человека одним сообщением, проверить распознанный черновик и подтвердить добавление в архив."
}

func formatUptime(uptime time.Duration) string {
	if uptime < 0 {
		uptime = 0
	}

	uptime = uptime.Round(time.Second)
	days := int(uptime / (24 * time.Hour))
	uptime -= time.Duration(days) * 24 * time.Hour
	hours := int(uptime / time.Hour)
	uptime -= time.Duration(hours) * time.Hour
	minutes := int(uptime / time.Minute)
	uptime -= time.Duration(minutes) * time.Minute
	seconds := int(uptime / time.Second)

	parts := make([]string, 0, 4)
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%d д", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%d ч", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%d мин", minutes))
	}
	if seconds > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%d сек", seconds))
	}

	return strings.Join(parts, " ")
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
