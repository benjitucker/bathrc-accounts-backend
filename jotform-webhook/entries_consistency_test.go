package jotform_webhook

import (
	"encoding/json"
	"fmt"
	"testing"
)

// session describes one session of a booking, for building both forms of a submission.
type session struct {
	membership, horse, venue, date          string
	sessionQID, venueQID, membQID, horseQID int
}

// webhookJSON builds the webhook's rawRequest, whose keys are "q<id>_<unique name>".
func webhookJSON(sessions []session) []byte {
	m := map[string]any{"submitDate": "1793820597695", "q58_totalAmount": "0", "q12_typeA": "ZFYA"}
	for i, s := range sessions {
		suffix := ""
		if i > 0 {
			suffix = "-2"
		}
		m[fmt.Sprintf("q%d_brcMembership15%s", s.membQID, suffix)] = s.membership
		m[fmt.Sprintf("q%d_horseName18%s", s.horseQID, suffix)] = s.horse
		m[fmt.Sprintf("q%d_selectedVenue%s", s.venueQID, suffix)] = s.venue
		m[fmt.Sprintf("q%d_select%sSession%s", s.sessionQID, s.venue, suffix)] = map[string]string{
			"date": s.date, "duration": "60", "timezone": "Europe/London (GMT+01:00)"}
	}
	b, _ := json.Marshal(m)
	return b
}

// apiJSON builds the Jotform API's version, whose answers are keyed by question id.
func apiJSON(sessions []session) []byte {
	answers := map[string]any{}
	add := func(qid int, name string, answer any) {
		answers[fmt.Sprint(qid)] = map[string]any{"name": name, "type": "control_textbox", "answer": answer}
	}
	for i, s := range sessions {
		suffix := ""
		if i > 0 {
			suffix = "-2"
		}
		add(s.membQID, "brcMembership15"+suffix, s.membership)
		add(s.horseQID, "horseName18"+suffix, s.horse)
		add(s.venueQID, "selectedVenue"+suffix, s.venue)
		add(s.sessionQID, "select"+s.venue+"Session"+suffix, map[string]string{"date": s.date, "duration": "60", "timezone": "Europe/London"})
	}
	b, _ := json.Marshal(map[string]any{"id": "1", "form_id": "263062182377357", "created_at": "2026-11-05 10:00:00", "answers": answers})
	return b
}

func first(membership, horse string) session {
	return session{membership: membership, horse: horse, venue: "WestWilts", date: "2026-11-12 19:00",
		sessionQID: 5, venueQID: 61, membQID: 15, horseQID: 18}
}

func second(membership, horse, venue string) session {
	return session{membership: membership, horse: horse, venue: venue, date: "2026-11-13 18:00",
		sessionQID: 64, venueQID: 63, membQID: 48, horseQID: 50}
}

// Both ways of reading a submission must agree on which sessions it contains, because whichever
// arrives first (webhook, redirect or hourly catch-up) is the one that is processed.
func TestWebhookAndAPIReadTheSameSessions(t *testing.T) {
	tests := []struct {
		name     string
		sessions []session
		want     []string // membership numbers of the sessions found
	}{
		{"one session", []session{first("111", "Ben")}, []string{"111"}},
		{"two sessions", []session{first("111", "Ben"), second("222", "Star", "Widbrook")}, []string{"111", "222"}},
		{"second session at Stockley Farm", []session{first("111", "Ben"), second("222", "Star", "StockleyFarm")}, []string{"111", "222"}},
		{"second session with a membership number but no horse", []session{first("111", "Ben"), second("222", "", "Widbrook")}, []string{"111", "222"}},
		{"second session with a horse but no membership number", []session{first("111", "Ben"), second("", "Star", "Widbrook")}, []string{"111"}},
		{"empty second session", []session{first("111", "Ben"), second("", "", "")}, []string{"111"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var webhook TrainingRawRequest
			if err := json.Unmarshal(webhookJSON(tt.sessions), &webhook); err != nil {
				t.Fatalf("webhook: %v", err)
			}
			var api TrainingRawRequestWithID
			if err := json.Unmarshal(apiJSON(tt.sessions), &api); err != nil {
				t.Fatalf("api: %v", err)
			}

			for source, entries := range map[string][]Entry{"webhook": webhook.Entries, "api": api.Entries} {
				var got []string
				for _, e := range entries {
					got = append(got, e.MembershipNumber)
				}
				if fmt.Sprint(got) != fmt.Sprint(tt.want) {
					t.Errorf("%s found sessions %v, want %v", source, got, tt.want)
				}
			}

			// Both must also find the same venue and a session date for each session
			for i := range webhook.Entries {
				if i >= len(api.Entries) {
					break
				}
				w, a := webhook.Entries[i], api.Entries[i]
				if w.Venue != a.Venue {
					t.Errorf("session %d venue: webhook %q, api %q", i+1, w.Venue, a.Venue)
				}
				if w.SelectSession.StartLocal.IsZero() || a.SelectSession.StartLocal.IsZero() {
					t.Errorf("session %d has no date (webhook %v, api %v)", i+1, w.SelectSession.StartLocal, a.SelectSession.StartLocal)
				}
			}
		})
	}
}
