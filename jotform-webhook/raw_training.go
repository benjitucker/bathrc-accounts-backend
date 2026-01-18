package jotform_webhook

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type sessionJSON struct {
	Implementation string `json:"implementation"`
	Date           string `json:"date"`
	Duration       string `json:"duration"`
	Timezone       string `json:"timezone"`
}

type Session struct {
	StartLocal time.Time
	Duration   time.Duration
	Timezone   string
}

type Entry struct {
	MembershipNumber           string
	CurrentMembershipSelection []string
	HorseName                  string
	SelectSession              Session
	Venue                      string
	Amount                     string
}

type TrainingRawRequest struct {
	SubmitDate       UnixMillis `json:"submitDate"`
	BuildDate        UnixMillis `json:"buildDate"`
	PaymentRef       string     `json:"q53_paymentRef"`
	PaymentReference string     `json:"q12_typeA"`
	TotalAmount      string     `json:"q58_totalAmount"`

	Entries []Entry `json:"-"`
}

func (r *TrainingRawRequest) UnmarshalJSON(b []byte) error {
	type alias TrainingRawRequest
	aux := (*alias)(r)

	// First unmarshal standard fields
	if err := json.Unmarshal(b, aux); err != nil {
		return err
	}

	// Then parse the entry fields
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}

	r.Entries = []Entry{}

	for i := 0; ; i++ {
		suffix := ""
		if i > 0 {
			suffix = fmt.Sprintf("-%d", i+1)
		}

		entry := Entry{}
		hasEntry := false

		for k, v := range m {
			if strings.HasSuffix(k, "horseName18"+suffix) {
				_ = json.Unmarshal(v, &entry.HorseName)
			}

			if strings.HasSuffix(k, "brcMembership15"+suffix) {
				_ = json.Unmarshal(v, &entry.MembershipNumber)
				// If it's an empty membership number string, there is no entry here.
				if entry.MembershipNumber == "" {
					break
				}
				hasEntry = true
			}

			if strings.HasSuffix(k, "typeA28"+suffix) {
				_ = json.Unmarshal(v, &entry.CurrentMembershipSelection)
			}

			if strings.HasSuffix(k, "amount"+suffix) {
				_ = json.Unmarshal(v, &entry.Amount)
			}

			if strings.HasSuffix(k, "selectedVenue"+suffix) {
				_ = json.Unmarshal(v, &entry.Venue)
			}
		}

		// Use the venue from the first pass to find the session selection
		for k, v := range m {
			if strings.HasSuffix(k, "select"+entry.Venue+"Session"+suffix) {
				var sess sessionJSON
				_ = json.Unmarshal(v, &sess)

				if sess.Date != "" {
					// Parse session date+timezone
					start, err := ParseSessionDate(sess.Date, sess.Timezone)
					if err != nil {
						return fmt.Errorf("session %d date parse failed: %w", i+1, err)
					}

					var mins int
					fmt.Sscan(sess.Duration, &mins)
					entry.SelectSession = Session{
						StartLocal: start,
						Duration:   time.Duration(mins) * time.Minute,
						Timezone:   sess.Timezone,
					}
				}
			}
		}

		if !hasEntry {
			// Exit here, no more entries
			return nil
		}

		r.Entries = append(r.Entries, entry)
	}
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

func (TrainingRawRequest) FormKind() string {
	return "Training"
}
