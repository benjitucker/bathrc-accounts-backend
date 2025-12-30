package jotform_webhook

import (
	"encoding/base64"
	"testing"
)

func mustBase64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func TestDecodeBase64Multipart_Training(t *testing.T) {
	payload := `--------------------------boundary
Content-Disposition: form-data; name="formTitle"

Training
--------------------------boundary
Content-Disposition: form-data; name="submissionID"

123
--------------------------boundary
Content-Disposition: form-data; name="rawRequest"

{"submitDate":"1765134783857","buildDate":"1765134764914","q15_brcMembership15":"1234567","q18_horseName18":"luke","q5_selectSession":{"implementation":"new","date":"2025-12-11 20:00","duration":"60","timezone":"Europe/London"},"q34_selectedVenue":"West Wilts","q12_typeA":"ZL44","q31_amount":"26"}
--------------------------boundary--`

	form, err := DecodeBase64Multipart(mustBase64(payload))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if form.FormTitle != "Training" {
		t.Fatalf("expected Training, got %s", form.FormTitle)
	}

	rr, ok := form.RawRequest.(TrainingRawRequest)
	if !ok {
		t.Fatalf("rawRequest type mismatch")
	}

	if rr.Entries[0].MembershipNumber != "1234567" {
		t.Errorf("membership mismatch")
	}

	if rr.Entries[0].SelectSession.Duration != 60 {
		t.Errorf("duration not parsed")
	}

	if rr.Entries[0].SelectSession.StartLocal.IsZero() {
		t.Errorf("session date not parsed")
	}
}
