package jotform

import (
	"encoding/json"
	"testing"
)

func TestTrainingAdminRawRequest_Unmarshal(t *testing.T) {
	js := `{
		"submitDate":"1765736311205",
		"buildDate":"1765736298125",
		"q7_typeA":"OFF",
		"uploadStatement":["file.csv"]
	}`

	var rr TrainingAdminRawRequest
	if err := json.Unmarshal([]byte(js), &rr); err != nil {
		t.Fatal(err)
	}

	if rr.SendEmailsNow != "OFF" {
		t.Errorf("SendEmailsNow mismatch")
	}

	if len(rr.UploadURLs) != 1 {
		t.Errorf("UploadURLs not parsed")
	}
}
