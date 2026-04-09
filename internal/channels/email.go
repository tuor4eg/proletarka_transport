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
	to       string
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

	body := buildEmailBody(c.from, c.to, message.Subject, message.Text)
	if err := smtp.SendMail(addr, auth, c.from, []string{c.to}, []byte(body)); err != nil {
		return fmt.Errorf("send email message: %w", err)
	}

	return nil
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
