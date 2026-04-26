package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

type PersonDraft struct {
	Person   DraftPerson  `json:"person"`
	Events   []DraftEvent `json:"events"`
	Warnings []string     `json:"warnings"`
}

type DraftPerson struct {
	Name       string  `json:"name"`
	ShortBio   *string `json:"shortBio"`
	BirthYear  *int    `json:"birthYear"`
	DeathYear  *int    `json:"deathYear"`
	YearsLabel *string `json:"yearsLabel"`
}

type DraftEvent struct {
	Text       string   `json:"text"`
	YearFrom   *int     `json:"yearFrom"`
	YearTo     *int     `json:"yearTo"`
	YearsLabel *string  `json:"yearsLabel"`
	TopicCodes []string `json:"topicCodes"`
}

func BuildPersonDraftInput(topics json.RawMessage, source string) string {
	topics = json.RawMessage(strings.TrimSpace(string(topics)))
	if len(topics) == 0 {
		topics = json.RawMessage("[]")
	}

	return strings.Join([]string{
		"topics:",
		string(topics),
		"",
		"source_text:",
		strings.TrimSpace(source),
	}, "\n")
}

func ParsePersonDraft(raw string) (PersonDraft, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return PersonDraft{}, fmt.Errorf("person draft response is empty")
	}

	var draft PersonDraft
	if err := json.Unmarshal([]byte(raw), &draft); err != nil {
		return PersonDraft{}, fmt.Errorf("decode person draft response: %w", err)
	}

	return draft, nil
}

func FormatPersonDraft(draft PersonDraft, topicTitles map[string]string) string {
	var builder strings.Builder
	builder.WriteString("Черновик для проверки\n\n")

	name := strings.TrimSpace(draft.Person.Name)
	if name == "" {
		name = "не указано"
	}
	builder.WriteString("Человек: ")
	builder.WriteString(name)
	builder.WriteByte('\n')

	if years := formatPersonYears(draft.Person); years != "" {
		builder.WriteString("Годы жизни: ")
		builder.WriteString(years)
		builder.WriteByte('\n')
	}

	if draft.Person.ShortBio != nil && strings.TrimSpace(*draft.Person.ShortBio) != "" {
		builder.WriteString("\nКраткое описание:\n")
		builder.WriteString(strings.TrimSpace(*draft.Person.ShortBio))
		builder.WriteByte('\n')
	}

	if len(draft.Events) > 0 {
		builder.WriteString("\nСобытия:\n")
		for _, event := range draft.Events {
			text := strings.TrimSpace(event.Text)
			if text == "" {
				continue
			}

			builder.WriteString("- ")
			if years := formatEventYears(event); years != "" {
				builder.WriteString(years)
				builder.WriteString(": ")
			}
			builder.WriteString(text)
			if len(event.TopicCodes) > 0 {
				builder.WriteString(" [темы: ")
				builder.WriteString(strings.Join(formatTopicNames(event.TopicCodes, topicTitles), ", "))
				builder.WriteString("]")
			}
			builder.WriteByte('\n')
		}
	}

	if len(draft.Warnings) > 0 {
		builder.WriteString("\nПредупреждения:\n")
		for _, warning := range draft.Warnings {
			warning = strings.TrimSpace(warning)
			if warning == "" {
				continue
			}
			builder.WriteString("- ")
			builder.WriteString(warning)
			builder.WriteByte('\n')
		}
	}

	return strings.TrimSpace(builder.String())
}

func formatTopicNames(codes []string, titles map[string]string) []string {
	result := make([]string, 0, len(codes))
	seen := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}

		title := strings.TrimSpace(titles[code])
		if title == "" {
			title = code
		}
		result = append(result, title)
	}

	return result
}

func formatPersonYears(person DraftPerson) string {
	if person.YearsLabel != nil && strings.TrimSpace(*person.YearsLabel) != "" {
		return strings.TrimSpace(*person.YearsLabel)
	}

	if person.BirthYear == nil && person.DeathYear == nil {
		return ""
	}
	if person.BirthYear != nil && person.DeathYear != nil {
		return fmt.Sprintf("%d-%d", *person.BirthYear, *person.DeathYear)
	}
	if person.BirthYear != nil {
		return fmt.Sprintf("%d-", *person.BirthYear)
	}

	return fmt.Sprintf("-%d", *person.DeathYear)
}

func formatEventYears(event DraftEvent) string {
	if event.YearsLabel != nil && strings.TrimSpace(*event.YearsLabel) != "" {
		return strings.TrimSpace(*event.YearsLabel)
	}

	if event.YearFrom == nil && event.YearTo == nil {
		return ""
	}
	if event.YearFrom != nil && event.YearTo != nil {
		return fmt.Sprintf("%d-%d", *event.YearFrom, *event.YearTo)
	}
	if event.YearFrom != nil {
		return fmt.Sprintf("%d", *event.YearFrom)
	}

	return fmt.Sprintf("до %d", *event.YearTo)
}
