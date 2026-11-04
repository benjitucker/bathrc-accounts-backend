package email

import (
	"context"
	"strings"
	"testing"
)

func TestPaymentExpiredEmail(t *testing.T) {
	eh, err := NewEmailHandler(context.Background(), nil, HandlerParams{})
	if err != nil {
		t.Fatalf("loading templates: %v", err)
	}
	subject, html, text, err := eh.Render("payment-expired", &PaymentExpiredData{
		FirstName: "Alice", Sessions: "a training session at WestWilts on Thursday 12th November at 7:00 PM",
		Deadline: "3:45 PM on Thursday 5th November"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if subject != "Training request cancelled" {
		t.Errorf("subject %q", subject)
	}
	for format, body := range map[string]string{"html": html, "text": text} {
		for _, want := range []string{"Alice", "WestWilts", "3:45 PM on Thursday 5th November", "released", "book again"} {
			if !strings.Contains(body, want) {
				t.Errorf("%s is missing %q", format, want)
			}
		}
	}
}

func TestReceivedRequest_ShowsPaymentDeadline(t *testing.T) {
	eh, _ := NewEmailHandler(context.Background(), nil, HandlerParams{})
	_, html, text, err := eh.Render("received-request", &ReceivedRequestData{
		Amount: "21.00", PayDate: "Tuesday 10th November", PayLink: "https://x/pay?s=1&t=2",
		PayDeadline: "3:45 PM on Thursday 5th November"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for format, body := range map[string]string{"html": html, "text": text} {
		if !strings.Contains(body, "3:45 PM on Thursday 5th November") || !strings.Contains(body, "will be released") {
			t.Errorf("%s: deadline wording missing", format)
		}
		if strings.Contains(body, "Tuesday 10th November") {
			t.Errorf("%s: the old pay-by date should not be shown with a deadline", format)
		}
	}
}
