package email

import (
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"path/filepath"
	"strings"
	texttmpl "text/template"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

const (
	sender   = "training@bathridingclub.co.uk"
	testMode = true
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
	if testMode {
		recipient = "ben@churchfarmmonktonfarleigh.co.uk"
	}

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

func (eh *EmailHandler) SendEmailPretty(recipient, templateName string, templateData any) {
	if testMode {
		recipient = "ben@churchfarmmonktonfarleigh.co.uk"
	}

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
	encodedLogo := base64.StdEncoding.EncodeToString(logoBytes)

	// Replace placeholder in HTML if needed
	// Ensure your template has: <img src="cid:logo123" alt="Logo">
	// htmlBody already references cid:logo123

	// --- Build Raw MIME email ---
	var raw bytes.Buffer
	mixed := multipart.NewWriter(&raw)

	raw.WriteString(fmt.Sprintf("From: %s\r\n", sender))
	raw.WriteString(fmt.Sprintf("To: %s\r\n", recipient))
	raw.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	raw.WriteString("MIME-Version: 1.0\r\n")
	raw.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", mixed.Boundary()))
	raw.WriteString("\r\n")

	// multipart/alternative for text + html
	altHeader := textproto.MIMEHeader{}
	altHeader.Set("Content-Type", fmt.Sprintf("multipart/alternative; boundary=\"alt-%s\"", mixed.Boundary()))
	altPart, _ := mixed.CreatePart(altHeader)
	alt := multipart.NewWriter(altPart)

	// text/plain
	txtHeader := textproto.MIMEHeader{}
	txtHeader.Set("Content-Type", "text/plain; charset=UTF-8")
	txtHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	txtWriter, _ := alt.CreatePart(txtHeader)
	qpTxt := quotedprintable.NewWriter(txtWriter)
	qpTxt.Write([]byte(textBody))
	qpTxt.Close()

	// html + inline image (multipart/related)
	relHeader := textproto.MIMEHeader{}
	relHeader.Set("Content-Type", fmt.Sprintf("multipart/related; boundary=\"rel-%s\"", mixed.Boundary()))
	relPart, _ := alt.CreatePart(relHeader)
	rel := multipart.NewWriter(relPart)

	htmlHeader := textproto.MIMEHeader{}
	htmlHeader.Set("Content-Type", "text/html; charset=UTF-8")
	htmlHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	htmlWriter, _ := rel.CreatePart(htmlHeader)
	qpHtml := quotedprintable.NewWriter(htmlWriter)
	qpHtml.Write([]byte(htmlBody))
	qpHtml.Close()

	// Inline image
	imgHeader := textproto.MIMEHeader{}
	imgHeader.Set("Content-Type", "image/png")
	imgHeader.Set("Content-Transfer-Encoding", "base64")
	imgHeader.Set("Content-ID", "<logo123>")
	imgHeader.Set("Content-Disposition", "inline; filename=\"logo.png\"")
	imgWriter, _ := rel.CreatePart(imgHeader)
	imgWriter.Write([]byte(encodedLogo))

	rel.Close()
	alt.Close()
	mixed.Close()

	input := &ses.SendRawEmailInput{
		Source:       aws.String(sender), // sender
		Destinations: []string{recipient},
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
