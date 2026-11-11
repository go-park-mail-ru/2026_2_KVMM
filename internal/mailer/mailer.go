package mailer

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"net/smtp"
	"os"
	"strings"
	"sync"
	"time"
)

//go:embed templates/auth.html
var templateFS embed.FS

const emailCooldown = 2 * time.Minute

var ErrCooldown = errors.New("authorization code can only be sent once every 2 minutes")

// EmailData содержит данные для шаблона письма
type EmailData struct {
	Code    string
	LogoURL string
}

// Mailer отправляет письма с кодами подтверждения через SMTP
type Mailer struct {
	mu       sync.Mutex
	lastSent map[string]time.Time
}

// New создаёт SMTP-mailer
func New() *Mailer {
	return &Mailer{lastSent: make(map[string]time.Time)}
}

// SendAuthCode отправляет одноразовый код на email
func (m *Mailer) SendAuthCode(to, code string) error {
	to = strings.ToLower(strings.TrimSpace(to))
	if !m.ReserveAuthCode(to) {
		return ErrCooldown
	}

	success := false
	defer func() {
		if !success {
			m.mu.Lock()
			delete(m.lastSent, to)
			m.mu.Unlock()
		}
	}()

	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	username := os.Getenv("SMTP_USERNAME")
	password := os.Getenv("SMTP_PASSWORD")
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	fromName := strings.TrimSpace(os.Getenv("SMTP_FROM_NAME"))
	logoURL := strings.TrimSpace(os.Getenv("LOGO_URL"))

	if host == "" || port == "" || from == "" || password == "" || fromName == "" {
		return errors.New("SMTP_HOST, SMTP_PORT and SMTP_FROM are required")
	}

	tmpl, err := template.ParseFS(templateFS, "templates/auth.html")
	if err != nil {
		return fmt.Errorf("parse email template: %w", err)
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, EmailData{
		Code:    code,
		LogoURL: logoURL,
	}); err != nil {
		return fmt.Errorf("render email template: %w", err)
	}

	msg := []byte(
		"From: " + fromName + " <" + from + ">\r\n" +
			"To: " + to + "\r\n" +
			"Subject: Ваш код подтверждения " + code + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n" +
			"\r\n" +
			body.String(),
	)

	auth := smtp.PlainAuth("", username, password, host)
	if err := smtp.SendMail(host+":"+port, auth, from, []string{to}, msg); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	success = true
	return nil
}

// ReserveAuthCode резервирует email на время cooldown без отправки письма
func (m *Mailer) ReserveAuthCode(to string) bool {
	return m.reserve(strings.ToLower(strings.TrimSpace(to)))
}

// reserve проверяет cooldown и резервирует email для отправки
func (m *Mailer) reserve(email string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sentAt, ok := m.lastSent[email]; ok && time.Since(sentAt) < emailCooldown {
		return false
	}
	m.lastSent[email] = time.Now()
	return true
}
