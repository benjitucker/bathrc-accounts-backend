package main

import (
	"benjitucker/bathrc-accounts/db"
	"strings"
	"testing"
)

// A form questions response in the shape the Jotform API returns: question id -> question.
const sampleQuestions = `{
  "5":  {"qid":"5",  "name":"selectWestWiltsSession", "type":"control_appointment", "text":"Select West Wilts Session Date and Preferred Time"},
  "54": {"qid":"54", "name":"wwecnonmem", "type":"control_textbox", "text":"WWEC-NON-MEM", "defaultValue":"26"},
  "55": {"qid":"55", "name":"wwecmem",    "type":"control_textbox", "text":"WWEC-MEM",     "defaultValue":"21"},
  "56": {"qid":"56", "name":"widnonmem",  "type":"control_textbox", "text":"WID-NON-MEM",  "defaultValue":"20"},
  "57": {"qid":"57", "name":"widmem",     "type":"control_textbox", "text":"WID-MEM",      "defaultValue":"16.50"},
  "68": {"qid":"68", "name":"slfnonmem",  "type":"control_textbox", "text":"SLF-NON-MEM",  "defaultValue":"£26"},
  "69": {"qid":"69", "name":"slfmem",     "type":"control_textbox", "text":"SLF-MEM",      "defaultValue":"21"},
  "70": {"qid":"70", "name":"RiverMeadMem",    "type":"control_textbox", "text":"RM-MEM",     "value":"18"},
  "71": {"qid":"71", "name":"rivermeadnonmem", "type":"control_textbox", "text":"RM-NON-MEM", "value":"23"},
  "72": {"qid":"72", "name":"brokenmem",       "type":"control_textbox", "text":"BROKEN-MEM"}
}`

func TestPricesForVenue(t *testing.T) {
	fields, err := parsePriceFields([]byte(sampleQuestions))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	tests := []struct {
		venue          string
		member, nonMem int64
		found          bool
	}{
		{"WestWilts", 2100, 2600, true},    // legacy prefix wwec
		{"Widbrook", 1650, 2000, true},     // legacy prefix wid, pence from "16.50"
		{"StockleyFarm", 2100, 2600, true}, // legacy prefix slf, "£26"
		{"RiverMead", 1800, 2300, true},    // new venue by convention, any case, "value" property
		{"rivermead", 1800, 2300, true},
		{"Broken", 0, 0, false},  // member field has no price, non-member field missing
		{"Unknown", 0, 0, false}, // no fields at all
		{"", 0, 0, false},
	}
	for _, tt := range tests {
		got, ok := fields.pricesForVenue(tt.venue)
		if ok != tt.found {
			t.Errorf("%q: found %v, want %v", tt.venue, ok, tt.found)
			continue
		}
		if ok && (got.MemberPence != tt.member || got.NonMemberPence != tt.nonMem) {
			t.Errorf("%q: got member %d non-member %d, want %d %d", tt.venue, got.MemberPence, got.NonMemberPence, tt.member, tt.nonMem)
		}
	}
}

func TestParsePriceFields_IgnoresOtherFields(t *testing.T) {
	fields, err := parsePriceFields([]byte(sampleQuestions))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if _, ok := fields.raw["selectwestwiltssession"]; ok {
		t.Errorf("non-price field included")
	}
	if _, ok := fields.raw["brokenmem"]; !ok {
		t.Errorf("a price field without a value should still be kept for the diagnostic")
	}
}

func TestParsePriceFields_InvalidJSON(t *testing.T) {
	if _, err := parsePriceFields([]byte("not json")); err == nil {
		t.Errorf("expected an error")
	}
}

func TestPriceDiagnostic(t *testing.T) {
	fields, _ := parsePriceFields([]byte(sampleQuestions))
	d := fields.diagnostic()
	for _, want := range []string{"wwecmem: 2100", "brokenmem: no price read", `"text":"BROKEN-MEM"`} {
		if !strings.Contains(d, want) {
			t.Errorf("diagnostic missing %q", want)
		}
	}

	empty, _ := parsePriceFields([]byte(`{"1":{"name":"horseName18"}}`))
	if !strings.Contains(empty.diagnostic(), "No fields") {
		t.Errorf("expected the no-fields message")
	}
}

func TestParsePoundsToPence(t *testing.T) {
	for in, want := range map[string]int64{"21": 2100, "21.5": 2150, "£16.50": 1650, " 0 ": 0, "19.999": 2000} {
		got, ok := parsePoundsToPence(in)
		if !ok || got != want {
			t.Errorf("parsePoundsToPence(%q) = %d, %v; want %d", in, got, ok, want)
		}
	}
	for _, bad := range []string{"", "abc", "-5", "NaN", "Inf"} {
		if _, ok := parsePoundsToPence(bad); ok {
			t.Errorf("parsePoundsToPence(%q) should fail", bad)
		}
	}
}

func TestChargeAmountPence(t *testing.T) {
	prices := func(b *db.TrainingSubmission) { b.MemberPricePence = 2100; b.NonMemberPricePence = 2600 }

	tests := []struct {
		name               string
		requested, current bool
		want               int64
	}{
		{"member at member price", true, true, 2100},
		{"asked for non-member price but is a member: member price", false, true, 2100},
		{"lapsed member after refresh: non-member price", true, false, 2600},
		{"non-member", false, false, 2600},
	}
	for _, tt := range tests {
		b := booking(prices)
		b.RequestCurrMem = tt.requested
		b.ActualCurrMem = tt.current
		if got := chargeAmountPence(b); got != tt.want {
			t.Errorf("%s: got %d, want %d", tt.name, got, tt.want)
		}
	}
}
