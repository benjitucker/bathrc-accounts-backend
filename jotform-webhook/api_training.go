package jotform_webhook

import (
	"encoding/json"
	"fmt"
	"time"
)

type apiSubmission struct {
	ID        string                    `json:"id"`
	FormID    string                    `json:"form_id"`
	IP        string                    `json:"ip"`
	CreatedAt string                    `json:"created_at"`
	Answers   map[string]apiAnswerField `json:"answers"`
}

type apiAnswerField struct {
	Name   string          `json:"name"`
	Type   string          `json:"type"`
	Answer json.RawMessage `json:"answer,omitempty"`
}

type TrainingRawRequestWithID struct {
	SubmissionID string `json:"id"`
	TrainingRawRequest
}

func (r *TrainingRawRequestWithID) UnmarshalJSON(b []byte) error {
	var sub apiSubmission
	if err := json.Unmarshal(b, &sub); err != nil {
		return err
	}

	r.SubmissionID = sub.ID

	created, _ := time.Parse("2006-01-02 15:04:05", sub.CreatedAt)
	r.SubmitDate = UnixMillis(created)

	// Build name lookup
	byName := map[string]apiAnswerField{}
	for _, f := range sub.Answers {
		if f.Name != "" {
			byName[f.Name] = f
		}
	}

	r.PaymentRef = extractAPIString(byName, "paymentRef")
	r.PaymentReference = extractAPIString(byName, "typeA")
	r.TotalAmount = extractAPIString(byName, "totalAmount")

	r.Entries = []Entry{}

	for i := 0; ; i++ {
		suffix := ""
		if i > 0 {
			suffix = fmt.Sprintf("-%d", i+1)
		}

		horse := extractAPIString(byName, "horseName18"+suffix)
		if horse == "" {
			if i == 0 {
				continue
			}
			break
		}

		memb := extractAPIString(byName, "brcMembership15"+suffix)
		venue := extractAPIString(byName, "selectedVenue"+suffix)
		amount := extractAPIString(byName, "amount"+suffix)

		var membType []string
		if f, ok := byName["typeA28"+suffix]; ok && len(f.Answer) > 0 {
			_ = json.Unmarshal(f.Answer, &membType)
		}

		var sess struct {
			Date     string `json:"date"`
			Duration string `json:"duration"`
			Timezone string `json:"timezone"`
		}
		if f, ok := byName["select"+venue+"Session"+suffix]; ok && len(f.Answer) > 0 {
			_ = json.Unmarshal(f.Answer, &sess)
		}

		start, err := ParseSessionDate(sess.Date, sess.Timezone)
		if err != nil {
			return fmt.Errorf("session parse failed: %w", err)
		}

		var mins int
		fmt.Sscan(sess.Duration, &mins)

		r.Entries = append(r.Entries, Entry{
			MembershipNumber:           memb,
			CurrentMembershipSelection: membType,
			HorseName:                  horse,
			Venue:                      venue,
			Amount:                     amount,
			SelectSession: Session{
				StartLocal: start,
				Duration:   time.Duration(mins) * time.Minute,
				Timezone:   sess.Timezone,
			},
		})
	}

	return nil
}

func extractAPIString(m map[string]apiAnswerField, name string) string {
	if f, ok := m[name]; ok && len(f.Answer) > 0 {
		var v string
		_ = json.Unmarshal(f.Answer, &v)
		return v
	}
	return ""
}
