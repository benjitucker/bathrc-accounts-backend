package email

import (
	"strings"
	"testing"
)

func TestDeliverTo_Production(t *testing.T) {
	eh := &EmailHandler{params: HandlerParams{}}
	to, subject, ok := eh.deliverTo([]string{"member@example.org"}, "Training")
	if !ok || len(to) != 1 || to[0] != "member@example.org" || subject != "Training" {
		t.Fatalf("production must send as addressed, got %v %q %v", to, subject, ok)
	}
}

func TestDeliverTo_TestModeOnlyEmailsTheTestAddress(t *testing.T) {
	eh := &EmailHandler{params: HandlerParams{TestMode: true, TestRecipient: "me@example.org"}}
	to, subject, ok := eh.deliverTo([]string{"member1@example.org", "member2@example.org"}, "Training")
	if !ok {
		t.Fatalf("expected the email to be sent to the test address")
	}
	if len(to) != 1 || to[0] != "me@example.org" {
		t.Fatalf("test mode must only email the test address, got %v", to)
	}
	if !strings.Contains(subject, "member1@example.org") || !strings.Contains(subject, "member2@example.org") || !strings.HasSuffix(subject, "Training") {
		t.Errorf("subject should name the intended recipients: %q", subject)
	}
}

func TestDeliverTo_TestModeWithoutTestAddressSendsNothing(t *testing.T) {
	eh := &EmailHandler{params: HandlerParams{TestMode: true}}
	to, _, ok := eh.deliverTo([]string{"member@example.org"}, "Training")
	if ok || len(to) != 0 {
		t.Fatalf("with no test address nothing may be sent, got %v %v", to, ok)
	}
}
