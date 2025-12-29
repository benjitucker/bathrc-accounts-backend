package jotform_webhook

import (
	"encoding/json"
	"fmt"
	"time"
)

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

	// JotForm format: "YYYY-MM-DD HH:MM"
	t, err := time.Parse("2006-01-02 15:04", aux.Date)
	if err != nil {
		return fmt.Errorf("invalid session date: %w", err)
	}
	s.Date = t

	// Convert duration to int
	if _, err := fmt.Sscan(aux.Duration, &s.Duration); err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}

	return nil
}

// StringList handles JSON that may be a string or an array of strings.
type StringList []string

func (s *StringList) UnmarshalJSON(data []byte) error {
	// Try single string first
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*s = []string{single}
		return nil
	}

	// Then try array of strings
	var list []string
	if err := json.Unmarshal(data, &list); err == nil {
		*s = list
		return nil
	}

	// Allow null -> empty list
	if string(data) == "null" {
		*s = nil
		return nil
	}

	return fmt.Errorf("StringList: value must be string or []string, got: %s", string(data))
}

type TrainingRawRequest struct {
	Slug                       string        `json:"slug"`
	SubmitSource               string        `json:"submitSource"`
	SubmitDate                 UnixMillis    `json:"submitDate"`
	BuildDate                  UnixMillis    `json:"buildDate"`
	MembershipNumber           string        `json:"q15_brcMembership15"`
	HorseName                  string        `json:"q18_horseName18"`
	SelectSession              SelectSession `json:"q5_selectSession"`
	CurrentMembershipSelection StringList    `json:"q28_typeA28"`
	SelectedVenue              string        `json:"q34_selectedVenue"`
	PaymentReference           string        `json:"q12_typeA"`
	Amount                     string        `json:"q31_amount"`
	Preview                    string        `json:"preview"`
	Path                       string        `json:"path"`
}

func (TrainingRawRequest) FormKind() string {
	return "Training"
}
