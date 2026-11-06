package email

import (
	"context"
	"strings"
	"testing"
)

func TestDisplayName(t *testing.T) {
	for in, want := range map[string]string{
		"OLIVIA": "Olivia", "olivia": "Olivia", "Olivia": "Olivia", "McDonald": "McDonald",
		"MARY-JANE": "Mary-Jane", "O'NEIL": "O'Neil", "ANNE MARIE": "Anne Marie", "": "",
	} {
		if got := DisplayName(in); got != want {
			t.Errorf("DisplayName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestVenueDisplayName(t *testing.T) {
	for in, want := range map[string]string{"WestWilts": "West Wilts", "StockleyFarm": "Stockley Farm", "Widbrook": "Widbrook", "ABCFarm": "ABC Farm"} {
		if got := VenueDisplayName(in); got != want {
			t.Errorf("VenueDisplayName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEmailsShowTidyNamesAndVenues(t *testing.T) {
	eh, _ := NewEmailHandler(context.Background(), nil, HandlerParams{})
	_, html, text, err := eh.Render("received-request2", &ReceivedRequest2Data{
		FirstName: "Benjamin and OLIVIA", Venue: "WestWilts", Venue2: "StockleyFarm", Amount: "47.00", AccountNumber: "1"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for format, body := range map[string]string{"html": html, "text": text} {
		for _, want := range []string{"Benjamin and Olivia", "West Wilts", "Stockley Farm"} {
			if !strings.Contains(body, want) {
				t.Errorf("%s missing %q", format, want)
			}
		}
		for _, unwanted := range []string{"OLIVIA", "WestWilts", "StockleyFarm"} {
			if strings.Contains(body, unwanted) {
				t.Errorf("%s still contains %q", format, unwanted)
			}
		}
	}
}

func TestDuplicate2EmailNamesTheEarlierSession(t *testing.T) {
	eh, _ := NewEmailHandler(context.Background(), nil, HandlerParams{})
	for _, name := range []string{"received-duplicate-request2", "received-duplicate-paid-request2"} {
		_, html, text, err := eh.Render(name, &ReceivedRequest2Data{
			FirstName: "Benjamin", Venue: "WestWilts", TrainingDate: "Thursday 12th November at 6:00 PM",
			Venue2: "WestWilts", TrainingDate2: "Sunday 8th November at 6:00 PM",
			DuplicateSessions: "West Wilts on Sunday 8th November at 6:00 PM"})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for format, body := range map[string]string{"html": html, "text": text} {
			if !strings.Contains(body, "West Wilts on Sunday 8th November at 6:00 PM") || !strings.Contains(body, "Neither session in this request has been booked") {
				t.Errorf("%s %s: duplicate wording missing", name, format)
			}
		}
	}
}
