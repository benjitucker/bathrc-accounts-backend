package jotform_webhook

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"strconv"
	"strings"
)

func DecodeBase64Multipart(base64Payload string) (*FormData, error) {
	raw, err := base64.StdEncoding.DecodeString(base64Payload)
	if err != nil {
		return nil, err
	}

	idx := bytes.IndexByte(raw, '\n')
	if idx == -1 {
		return nil, fmt.Errorf("invalid multipart payload")
	}

	boundary := strings.TrimPrefix(strings.TrimSpace(string(raw[:idx])), "--")
	reader := multipart.NewReader(bytes.NewReader(raw), boundary)

	form := &FormData{}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		val, _ := io.ReadAll(part)
		value := string(val)

		switch part.FormName() {
		case "action":
			form.Action = value
		case "webhookURL":
			form.WebhookURL = value
		case "username":
			form.Username = value
		case "formID":
			form.FormID = value
		case "formTitle":
			form.FormTitle = value
		case "submissionID":
			form.SubmissionID = value
		case "pretty":
			form.Pretty = value
		case "ip":
			form.IP = value
		case "rawRequest":
			form.RawRequestStr = value
		}
	}

	switch form.FormTitle {
	case "Training":
		var rr TrainingRawRequest
		if err := json.Unmarshal([]byte(form.RawRequestStr), &rr); err != nil {
			return nil, err
		}
		form.RawRequest = rr

		// Validate the total amount
		total := 0.0
		for _, entry := range rr.Entries {
			amount, _ := strconv.ParseFloat(entry.Amount, 64)
			total = total + amount
		}
		totalAmount, _ := strconv.ParseFloat(rr.TotalAmount, 64)
		if total != totalAmount {
			return nil, fmt.Errorf("inconsistent amounts in training form: %f != %f",
				total, totalAmount)
		}

	case "Training Administration":
		var rr TrainingAdminRawRequest
		if err := json.Unmarshal([]byte(form.RawRequestStr), &rr); err != nil {
			return nil, err
		}
		form.RawRequest = rr

	default:
		return nil, fmt.Errorf("unsupported formTitle: %s", form.FormTitle)
	}

	return form, nil
}
