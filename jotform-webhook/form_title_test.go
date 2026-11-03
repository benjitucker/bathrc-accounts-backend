package jotform_webhook

import (
	"strings"
	"testing"
)

func trainingPayload(title string) string {
	return `--------------------------boundary
Content-Disposition: form-data; name="formTitle"

` + title + `
--------------------------boundary
Content-Disposition: form-data; name="submissionID"

123
--------------------------boundary
Content-Disposition: form-data; name="rawRequest"

{"submitDate":"1765134783857","buildDate":"1765134764914","q15_brcMembership15":"1234567","q18_horseName18":"luke","q5_selectWestWiltsSession":{"implementation":"new","date":"2025-12-11 20:00","duration":"60","timezone":"Europe/London"},"q34_selectedVenue":"WestWilts","q12_typeA":"ZL44","q31_amount":"26","q58_totalAmount": "26"}
--------------------------boundary--`
}

func adminPayload(title string) string {
	return `--------------------------boundary
Content-Disposition: form-data; name="formTitle"

` + title + `
--------------------------boundary
Content-Disposition: form-data; name="submissionID"

456
--------------------------boundary
Content-Disposition: form-data; name="rawRequest"

{"submitDate":"1765134783857"}
--------------------------boundary--`
}

func withSuffix(t *testing.T, suffix string) {
	SetFormTitleSuffix(suffix)
	t.Cleanup(func() { SetFormTitleSuffix("") })
}

func TestFormTitle_NoSuffix_RejectsDevForm(t *testing.T) {
	withSuffix(t, "")

	_, err := DecodeBase64Multipart(mustBase64(trainingPayload("Training DEV")))
	if err == nil || !strings.Contains(err.Error(), "unsupported formTitle") {
		t.Fatalf("expected unsupported formTitle error, got %v", err)
	}
}

func TestFormTitle_WithSuffix_AcceptsDevTrainingForm(t *testing.T) {
	withSuffix(t, " DEV")

	form, err := DecodeBase64Multipart(mustBase64(trainingPayload("Training DEV")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if form.FormTitle != "Training DEV" {
		t.Errorf("expected original title to be kept, got %q", form.FormTitle)
	}
	if _, ok := form.RawRequest.(TrainingRawRequest); !ok {
		t.Fatalf("expected TrainingRawRequest, got %T", form.RawRequest)
	}
	if form.RawRequest.FormKind() != "Training" {
		t.Errorf("expected form kind Training, got %q", form.RawRequest.FormKind())
	}
}

func TestFormTitle_WithSuffix_AcceptsDevAdminForm(t *testing.T) {
	withSuffix(t, " DEV")

	form, err := DecodeBase64Multipart(mustBase64(adminPayload("Training Administration DEV")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := form.RawRequest.(TrainingAdminRawRequest); !ok {
		t.Fatalf("expected TrainingAdminRawRequest, got %T", form.RawRequest)
	}
}

func TestFormTitle_WithSuffix_RejectsProductionForm(t *testing.T) {
	withSuffix(t, " DEV")

	_, err := DecodeBase64Multipart(mustBase64(trainingPayload("Training")))
	if err == nil || !strings.Contains(err.Error(), "unsupported formTitle") {
		t.Fatalf("expected the production form to be rejected, got %v", err)
	}
}
