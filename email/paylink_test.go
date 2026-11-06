package email

import (
	"context"
	"strings"
	"testing"
)

// Every email that shows bank details must show a payment link instead when it is given one.
func TestTemplates_PayLinkReplacesBankDetails(t *testing.T) {
	eh, err := NewEmailHandler(context.Background(), nil, HandlerParams{})
	if err != nil {
		t.Fatalf("loading templates: %v", err)
	}

	link := "https://abc.lambda-url.eu-west-3.on.aws/pay?s=6696&t=abc"
	cases := map[string]func(payLink string) any{
		"received-request": func(l string) any {
			return &ReceivedRequestData{AccountNumber: "12345678", Amount: "21.00", PayLink: l}
		},
		"received-request2": func(l string) any {
			return &ReceivedRequest2Data{AccountNumber: "12345678", Amount: "47.00", PayLink: l}
		},
		"pay-reminder":  func(l string) any { return &PayReminderData{AccountNumber: "12345678", Amount: "21.00", PayLink: l} },
		"pay-reminder2": func(l string) any { return &PayReminder2Data{AccountNumber: "12345678", Amount: "47.00", PayLink: l} },
		"received-duplicate-request": func(l string) any {
			return &ReceivedRequestData{AccountNumber: "12345678", Amount: "21.00", PayLink: l}
		},
	}

	for name, data := range cases {
		_, html, text, err := eh.Render(name, data(""))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for format, body := range map[string]string{"html": html, "text": text} {
			if !strings.Contains(body, "BACS") || !strings.Contains(body, "12345678") {
				t.Errorf("%s %s: bank details missing without a payment link", name, format)
			}
		}

		_, html, text, err = eh.Render(name, data(link))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for format, body := range map[string]string{"html": html, "text": text} {
			if strings.Contains(body, "BACS") || strings.Contains(body, "12345678") {
				t.Errorf("%s %s: bank details shown with a payment link", name, format)
			}
			if !strings.Contains(body, "Please pay") || !strings.Contains(body, "£") {
				t.Errorf("%s %s: payment request missing", name, format)
			}
		}
		if !strings.Contains(text, link) {
			t.Errorf("%s text: link missing", name)
		}
		if !strings.Contains(html, `href="https://abc.lambda-url.eu-west-3.on.aws/pay?s=6696&amp;t=abc"`) || !strings.Contains(html, "Pay now") {
			t.Errorf("%s html: Pay now button missing", name)
		}
	}
}
