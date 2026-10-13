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

type EmailData struct {
	Code string
}

var (
	emailMu   sync.Mutex
	emailSent = make(map[string]time.Time)
)

// reserveEmail блокирует email для отправки кода авторизации
func reserveEmail(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))

	emailMu.Lock()
	defer emailMu.Unlock()

	if _, exists := emailSent[email]; exists {
		return false
	}

	emailSent[email] = time.Now()
	return true
}

// releaseEmailAfterCooldown снимает блок с emial
func releaseEmailAfterCooldown(email string) {
	time.Sleep(emailCooldown)

	emailMu.Lock()
	delete(emailSent, email)
	emailMu.Unlock()
}

// SendAuthCode отправляет код на указанный email
func SendAuthCode(to string, code string) error {
	to = strings.ToLower(strings.TrimSpace(to))

	if !reserveEmail(to) {
		return errors.New("authorization code can only be sent once every 2 minutes")
	}

	success := false

	// Освобождает резерв email при неудачной отправке
	defer func() {
		if !success {
			emailMu.Lock()
			delete(emailSent, to)
			emailMu.Unlock()
		}
	}()

	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	username := os.Getenv("SMTP_USERNAME")
	password := os.Getenv("SMTP_PASSWORD")
	from := os.Getenv("SMTP_FROM")
	fromName := os.Getenv("SMTP_FROM_NAME")

	tmpl, err := template.ParseFS(templateFS, "templates/auth.html")
	if err != nil {
		return fmt.Errorf("parse email template: %w", err)
	}
	subject := fmt.Sprintf("Ваш код авторизации [%s]", code)

	var body bytes.Buffer

	if err := tmpl.Execute(&body, EmailData{
		Code: code,
	}); err != nil {
		return fmt.Errorf("render email template: %w", err)
	}

	msg := []byte(
		"From: " + fromName + " <" + from + ">\r\n" +
			"To: " + to + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n" +
			"\r\n" +
			body.String(),
	)

	auth := smtp.PlainAuth("", username, password, host)

	if err := smtp.SendMail(
		host+":"+port,
		auth,
		from,
		[]string{to},
		msg,
	); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	success = true

	go releaseEmailAfterCooldown(to)

	return nil
}
