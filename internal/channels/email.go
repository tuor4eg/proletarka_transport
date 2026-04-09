package channels

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"

	"proletarka_transport/internal/config"
)

type EmailChannel struct {
	host     string
	port     int
	user     string
	password string
	from     string
	to       []string
}

func NewEmailChannel(cfg config.EmailConfig) *EmailChannel {
	return &EmailChannel{
		host:     cfg.Host,
		port:     cfg.Port,
		user:     cfg.User,
		password: cfg.Password,
		from:     cfg.From,
		to:       cfg.To,
	}
}

func (c *EmailChannel) Name() string {
	return "email"
}

func (c *EmailChannel) Send(_ context.Context, message Message) error {
	auth := smtp.PlainAuth("", c.user, c.password, c.host)
	addr := fmt.Sprintf("%s:%d", c.host, c.port)

	var delivered int
	var failures []string

	for _, recipient := range c.to {
		body := buildEmailBody(c.from, recipient, message.Subject, message.Text)
		if err := smtp.SendMail(addr, auth, c.from, []string{recipient}, []byte(body)); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", recipient, err))
			continue
		}

		delivered++
	}

	if delivered > 0 {
		return nil
	}

	return fmt.Errorf("send email message to all recipients failed: %s", strings.Join(failures, "; "))
}

func buildEmailBody(from string, to string, subject string, text string) string {
	lines := []string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		text,
	}

	return strings.Join(lines, "\r\n")
}
