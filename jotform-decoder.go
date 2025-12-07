package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"strings"
	"time"
)

// -------------------- Time Types --------------------

// UnixMillis parses millisecond timestamps as strings into time.Time
type UnixMillis time.Time

func (t *UnixMillis) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	ms, err := parseInt64(s)
	if err != nil {
		return err
	}
	*t = UnixMillis(time.UnixMilli(ms))
	return nil
}

func (t UnixMillis) Time() time.Time {
	return time.Time(t)
}

func parseInt64(s string) (int64, error) {
	var i int64
	_, err := fmt.Sscan(s, &i)
	return i, err
}

// -------------------- RawRequest Struct --------------------

type SelectSession struct {
	Implementation string    `json:"implementation"`
	Date           time.Time `json:"date"`
	Duration       int       `json:"duration"`
	Timezone       string    `json:"timezone"`
}

func (s *SelectSession) UnmarshalJSON(b []byte) error {
	type alias SelectSession
	aux := &struct {
		Date     string `json:"date"`
		Duration string `json:"duration"`
		*alias
	}{
		alias: (*alias)(s),
	}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}

	// Parse date string "2006-01-02 15:04"
	parsed, err := time.Parse("2006-01-02 15:04", aux.Date)
	if err != nil {
		return err
	}
	s.Date = parsed

	// Convert duration to int
	var dur int
	_, err = fmt.Sscan(aux.Duration, &dur)
	if err != nil {
		return err
	}
	s.Duration = dur

	return nil
}

type RawRequest struct {
	Slug                         string                 `json:"slug"`
	JsExecutionTracker           string                 `json:"jsExecutionTracker"`
	SubmitSource                 string                 `json:"submitSource"`
	SubmitDate                   UnixMillis             `json:"submitDate"`
	BuildDate                    UnixMillis             `json:"buildDate"`
	UploadServerUrl              string                 `json:"uploadServerUrl"`
	EventObserver                string                 `json:"eventObserver"`
	MembershipNumber             string                 `json:"q15_brcMembership15"`
	HorseName                    string                 `json:"q18_horseName18"`
	SelectSession                SelectSession          `json:"q5_selectSession"`
	SelectedVenue                string                 `json:"q34_selectedVenue"`
	PaymentReference             string                 `json:"q12_typeA"`
	Amount                       string                 `json:"q31_amount"`
	TimeToSubmit                 string                 `json:"timeToSubmit"`
	Preview                      string                 `json:"preview"`
	ValidatedNewRequiredFieldIDs map[string]interface{} `json:"validatedNewRequiredFieldIDs"`
	Path                         string                 `json:"path"`
	TypeA28                      string                 `json:"q28_typeA28"`
}

// -------------------- FormData Struct --------------------

type FormData struct {
	Action        string     `json:"action"`
	WebhookURL    string     `json:"webhookURL"`
	Username      string     `json:"username"`
	FormID        string     `json:"formID"`
	Type          string     `json:"type"`
	CustomParams  string     `json:"customParams"`
	Product       string     `json:"product"`
	FormTitle     string     `json:"formTitle"`
	CustomTitle   string     `json:"customTitle"`
	SubmissionID  string     `json:"submissionID"`
	Event         string     `json:"event"`
	DocumentID    string     `json:"documentID"`
	TeamID        string     `json:"teamID"`
	Subject       string     `json:"subject"`
	IsSilent      string     `json:"isSilent"`
	CustomBody    string     `json:"customBody"`
	RawRequestStr string     `json:"rawRequest"`
	RawRequest    RawRequest `json:"-"`
	FromTable     string     `json:"fromTable"`
	AppID         string     `json:"appID"`
	Pretty        string     `json:"pretty"`
	Unread        string     `json:"unread"`
	Parent        string     `json:"parent"`
	IP            string     `json:"ip"`
}

// Pretty JSON string
func (f *FormData) String() string {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Sprintf("FormData{Error marshaling: %v}", err)
	}
	return string(data)
}

// Concise human-readable debug
func (f *FormData) DebugString() string {
	return fmt.Sprintf(
		"Form Submission Debug:\n"+
			"Action: %s\nUsername: %s\nFormID: %s\nSubmissionID: %s\nFormTitle: %s\nWebhookURL: %s\nIP: %s\n\n"+
			"Raw Request:\n"+
			"  Membership Number: %s\n"+
			"  Horse Name: %s\n"+
			"  Session: %s, %s (%d minutes)\n"+
			"  Selected Venue: %s\n"+
			"  Payment Reference: %s\n"+
			"  Amount: %s\n"+
			"  Preview: %s\n"+
			"  SubmitDate: %s\n"+
			"  BuildDate: %s\n",
		f.Action,
		f.Username,
		f.FormID,
		f.SubmissionID,
		f.FormTitle,
		f.WebhookURL,
		f.IP,
		f.RawRequest.MembershipNumber,
		f.RawRequest.HorseName,
		f.RawRequest.SelectSession.Date.Format(time.RFC1123),
		f.RawRequest.SelectSession.Timezone,
		f.RawRequest.SelectSession.Duration,
		f.RawRequest.SelectedVenue,
		f.RawRequest.PaymentReference,
		f.RawRequest.Amount,
		f.RawRequest.Preview,
		f.RawRequest.SubmitDate.Time().Format(time.RFC1123),
		f.RawRequest.BuildDate.Time().Format(time.RFC1123),
	)
}

// -------------------- Multipart Base64 Parser --------------------

func parseBase64Multipart(base64Data string) (*FormData, error) {
	rawData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return nil, fmt.Errorf("base64 decode error: %v", err)
	}

	firstLineEnd := bytes.IndexByte(rawData, '\n')
	if firstLineEnd == -1 {
		return nil, fmt.Errorf("cannot find boundary line")
	}

	boundary := strings.TrimSpace(string(rawData[:firstLineEnd]))
	boundary = strings.TrimPrefix(boundary, "--")

	reader := multipart.NewReader(bytes.NewReader(rawData), boundary)
	form := &FormData{}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		name := part.FormName()
		valueBytes, _ := io.ReadAll(part)
		value := string(valueBytes)

		switch name {
		case "action":
			form.Action = value
		case "webhookURL":
			form.WebhookURL = value
		case "username":
			form.Username = value
		case "formID":
			form.FormID = value
		case "type":
			form.Type = value
		case "customParams":
			form.CustomParams = value
		case "product":
			form.Product = value
		case "formTitle":
			form.FormTitle = value
		case "customTitle":
			form.CustomTitle = value
		case "submissionID":
			form.SubmissionID = value
		case "event":
			form.Event = value
		case "documentID":
			form.DocumentID = value
		case "teamID":
			form.TeamID = value
		case "subject":
			form.Subject = value
		case "isSilent":
			form.IsSilent = value
		case "customBody":
			form.CustomBody = value
		case "rawRequest":
			form.RawRequestStr = value
			_ = json.Unmarshal([]byte(value), &form.RawRequest)
		case "fromTable":
			form.FromTable = value
		case "appID":
			form.AppID = value
		case "pretty":
			form.Pretty = value
		case "unread":
			form.Unread = value
		case "parent":
			form.Parent = value
		case "ip":
			form.IP = value
		}
	}

	return form, nil
}
