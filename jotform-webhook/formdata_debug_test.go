package jotform_webhook

import (
	"strings"
	"testing"
)

func TestDebugString_Safe(t *testing.T) {
	fd := &FormData{
		FormTitle:    "Training",
		SubmissionID: "1",
		Username:     "test",
		IP:           "127.0.0.1",
		RawRequest: TrainingRawRequest{
			PaymentReference: "ABCD",
		},
	}

	out := fd.DebugString()
	if out == "" {
		t.Fatalf("DebugString returned empty")
	}

	if !strings.Contains(out, "Training") {
		t.Errorf("DebugString missing form type")
	}
}
