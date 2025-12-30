package email

import (
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"path/filepath"
	"strings"
	texttmpl "text/template"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

const (
	sender = "training@bathridingclub.co.uk"
)

//go:embed templates/*
var templateFS embed.FS

//go:embed assets/*
var assetsFS embed.FS

type EmailTemplates struct {
	HTML    *template.Template
	Text    *texttmpl.Template
	Subject string
}

type EmailHandler struct {
	ctx       context.Context
	sesClient *ses.Client
	templates map[string]EmailTemplates
}

func NewEmailHandler(ctx context.Context, sesClient *ses.Client) (*EmailHandler, error) {
	result := &EmailHandler{
		ctx:       ctx,
		sesClient: sesClient,
		templates: make(map[string]EmailTemplates),
	}

	entries, err := templateFS.ReadDir("templates")
	if err != nil {
		return result, err
	}

	groups := map[string]map[string]string{}

	for _, e := range entries {
		name := e.Name()
		parts := strings.Split(name, ".")
		if len(parts) < 3 {
			continue
		}

		key := parts[0]
		kind := parts[1]

		if groups[key] == nil {
			groups[key] = map[string]string{}
		}
		groups[key][kind] = filepath.Join("templates", name)
	}

	for key, m := range groups {
		var et EmailTemplates

		if subjFile, ok := m["subject"]; ok {
			b, err := templateFS.ReadFile(subjFile)
			if err != nil {
				return result, err
			}
			et.Subject = strings.TrimSpace(string(b))
		}

		if htmlFile, ok := m["html"]; ok {
			t, err := template.ParseFS(templateFS, htmlFile)
			if err != nil {
				return result, err
			}
			et.HTML = t
		}

		if txtFile, ok := m["txt"]; ok {
			t, err := texttmpl.ParseFS(templateFS, txtFile)
			if err != nil {
				return result, err
			}
			et.Text = t
		}

		result.templates[key] = et
	}

	return result, nil
}

func (eh *EmailHandler) SendEmail(recipient, subject, body string) {

	// Build the email input
	input := &ses.SendEmailInput{
		Destination: &types.Destination{
			ToAddresses: []string{recipient},
		},
		ReplyToAddresses: []string{"bathridingclub@hotmail.com"},
		Message: &types.Message{
			Subject: &types.Content{
				Charset: aws.String("UTF-8"),
				Data:    aws.String(subject),
			},
			Body: &types.Body{
				Text: &types.Content{
					Charset: aws.String("UTF-8"),
					Data:    aws.String(body),
				},
			},
		},
		Source: aws.String(sender),
	}

	// Send the email
	result, err := eh.sesClient.SendEmail(eh.ctx, input)
	if err != nil {
		log.Fatalf("failed to send email: %v", err)
	}

	fmt.Println("Email sent! Message ID:", *result.MessageId)
}

func (eh *EmailHandler) SendEmailPretty(recipients []string, templateName string, templateData any) {

	// Render templates
	subject, htmlBody, textBody, err := eh.Render(templateName, templateData)
	if err != nil {
		log.Fatal(err)
	}

	// Read inline logo
	logoBytes, err := assetsFS.ReadFile("assets/logo.png")
	if err != nil {
		log.Fatal(err)
	}

	// Replace placeholder in HTML if needed
	// Ensure your template has: <img src="cid:logo123" alt="Logo">
	// htmlBody already references cid:logo123

	// --- Build Raw MIME email ---

	// ---------- BOUNDARIES ----------
	mixedBoundary := "mixed_12345"
	altBoundary := "alt_12345"
	relBoundary := "rel_12345"

	var raw bytes.Buffer

	// ---------- HEADERS ----------
	raw.WriteString("From: " + sender + "\r\n")
	raw.WriteString("To: " + strings.Join(recipients, ", ") + "\r\n")
	raw.WriteString("Subject: " + subject + "\r\n")
	raw.WriteString("MIME-Version: 1.0\r\n")
	raw.WriteString("Content-Type: multipart/mixed; boundary=\"" + mixedBoundary + "\"\r\n")
	raw.WriteString("\r\n")

	// ---------- multipart/alternative ----------
	raw.WriteString("--" + mixedBoundary + "\r\n")
	raw.WriteString("Content-Type: multipart/alternative; boundary=\"" + altBoundary + "\"\r\n")
	raw.WriteString("\r\n")

	// text/plain
	raw.WriteString("--" + altBoundary + "\r\n")
	raw.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	raw.WriteString(textBody + "\r\n")

	// ---------- multipart/related ----------
	raw.WriteString("--" + altBoundary + "\r\n")
	raw.WriteString("Content-Type: multipart/related; boundary=\"" + relBoundary + "\"\r\n\r\n")

	// html body
	raw.WriteString("--" + relBoundary + "\r\n")
	raw.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
	raw.WriteString(htmlBody + "\r\n")

	// inline image
	raw.WriteString("--" + relBoundary + "\r\n")
	raw.WriteString("Content-Type: image/png\r\n")
	raw.WriteString("Content-Transfer-Encoding: base64\r\n")
	raw.WriteString("Content-ID: <logo123>\r\n")
	raw.WriteString("Content-Disposition: inline; filename=\"logo.png\"\r\n\r\n")

	// split base64 into lines (good practice)
	encoded := base64.StdEncoding.EncodeToString(logoBytes)
	for len(encoded) > 76 {
		raw.WriteString(encoded[:76] + "\r\n")
		encoded = encoded[76:]
	}
	raw.WriteString(encoded + "\r\n")

	// close related
	raw.WriteString("--" + relBoundary + "--\r\n")

	// close alternative
	raw.WriteString("--" + altBoundary + "--\r\n")

	// close mixed
	raw.WriteString("--" + mixedBoundary + "--\r\n")

	input := &ses.SendRawEmailInput{
		Source:       aws.String(sender), // sender
		Destinations: recipients,
		RawMessage: &types.RawMessage{
			Data: raw.Bytes(),
		},
	}

	// --- Send SES Raw Email ---
	_, err = eh.sesClient.SendRawEmail(eh.ctx, input)
	if err != nil {
		log.Fatalf("failed to send email: %v", err)
	}

	log.Println("Email sent with inline logo")
}

func (eh *EmailHandler) Render(templateName string, data any) (subject, html, text string, err error) {
	t, ok := eh.templates[templateName]
	if !ok {
		return "", "", "", fmt.Errorf("template %s not found", templateName)
	}

	subject = t.Subject

	if t.HTML != nil {
		var buf bytes.Buffer
		if err := t.HTML.Execute(&buf, data); err != nil {
			return "", "", "", err
		}
		html = buf.String()
	}

	if t.Text != nil {
		var buf bytes.Buffer
		if err := t.Text.Execute(&buf, data); err != nil {
			return "", "", "", err
		}
		text = buf.String()
	}

	return subject, html, text, nil
}

func formatCustomDate(t time.Time) string {
	hour := t.Hour() % 12
	if hour == 0 {
		hour = 12
	}
	minute := t.Minute()
	ampm := t.Format("PM")

	return fmt.Sprintf("%s %s %s at %d:%d%d %s",
		t.Format("Monday"),
		dayWithSuffix(t.Day()),
		t.Format("January"),
		hour, minute/10, minute%10,
		ampm,
	)
}

func dayWithSuffix(day int) string {
	if day >= 11 && day <= 13 {
		return fmt.Sprintf("%dth", day)
	}
	switch day % 10 {
	case 1:
		return fmt.Sprintf("%dst", day)
	case 2:
		return fmt.Sprintf("%dnd", day)
	case 3:
		return fmt.Sprintf("%drd", day)
	default:
		return fmt.Sprintf("%dth", day)
	}
}
