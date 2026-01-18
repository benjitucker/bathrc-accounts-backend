package jotform_webhook

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTrainingRawRequest_Unmarshal(t *testing.T) {
	js := `{
		"submitDate":"1765134783857",
		"buildDate":"1765134764914",
		"q15_brcMembership15":"ABC",
		"q18_horseName18":"Horse",
        "selectedVenue":"TestVenue",
		"q5_selectTestVenueSession":{
			"implementation":"new",
			"date":"2025-12-11 20:00",
			"duration":"45",
			"timezone":"Europe/London"
		}
	}`

	var rr TrainingRawRequest
	if err := json.Unmarshal([]byte(js), &rr); err != nil {
		t.Fatal(err)
	}

	if rr.Entries[0].SelectSession.Duration != time.Minute*45 {
		t.Errorf("duration parse failed")
	}

	startyear, _, _ := rr.Entries[0].SelectSession.StartLocal.Date()
	if startyear != 2025 {
		t.Errorf("date parse failed")
	}

	if rr.SubmitDate.Time().Before(time.Unix(0, 0)) {
		t.Errorf("submitDate invalid")
	}
}

func TestTrainingRawRequest_Unmarshal_one_entry(t *testing.T) {
	js := `{
  "slug": "submit/252725624662359",
  "jsExecutionTracker": "build-date-1767123676125=>init-started:1767123678849=>validator-called:1767123678890=>validator-mounted-false:1767123678891=>init-complete:1767123678894=>interval-complete:1767123699896=>onsubmit-fired:1767123739153=>observerSubmitHandler_received-submit-event:1767123739154=>submit-validation-passed:1767123739176=>observerSubmitHandler_validation-passed-submitting-form:1767123739202",
  "submitSource": "form",
  "submitDate": "1767123739202",
  "buildDate": "1767123676125",
  "uploadServerUrl": "https://upload.jotform.com/upload",
  "eventObserver": "1",
  "q15_brcMembership15": "1111111",
  "q18_horseName18": "Luke",
  "q5_selectWidbrookSession": {
    "implementation": "new",
    "date": "2026-01-02 20:00",
    "duration": "60",
    "timezone": "Europe/London (GMT)"
  },
  "q5_selectAltTestVenueSession": {
    "implementation": "new",
    "date": "2026-01-03 20:00",
    "duration": "60",
    "timezone": "Europe/London (GMT)"
  },
  "q34_selectedVenue": "Widbrook",
  "q31_amount": "20",
  "q58_totalAmount": "20",
  "q53_paymentRef": "CATJ",
  "q12_typeA": "CATJ",
  "q54_wwecnonmem": "26",
  "q55_wwecmem": "21",
  "q56_widnonmem": "20",
  "q57_widmem": "16",
  "q48_brcMembership15-2": "",
  "q50_horseName18-2": "",
  "q51_selectTestVenueSession-2": {
    "implementation": "new",
    "date": "",
    "duration": "60",
    "timezone": "Europe/London (GMT+00:00)"
  },
  "q51_selectAltTestVenueSession-2": {
    "implementation": "new",
    "date": "",
    "duration": "60",
    "timezone": "Europe/London (GMT+00:00)"
  },
  "q60_selectedVenue-2": "",
  "q59_amount-2": "0",
  "event_id": "1767123678850_252725624662359_TRJHlhC",
  "jotform_pwa": "1",
  "pwa_id": "253053519411349",
  "pwa_isPWA": "true",
  "pwa_device": "mobile",
  "timeToSubmit": "20",
  "validatedNewRequiredFieldIDs": "",
  "visitedPages": ""
}`
	var rr TrainingRawRequest
	if err := json.Unmarshal([]byte(js), &rr); err != nil {
		t.Fatal(err)
	}

	if len(rr.Entries) != 1 {
		t.Errorf("should be one entry")
	}

	_, _, startday := rr.Entries[0].SelectSession.StartLocal.Date()
	if startday != 2 {
		t.Errorf("date parse failed")
	}

	if rr.Entries[0].Venue != "Widbrook" {
		t.Errorf("venue parse failed")
	}

	if rr.SubmitDate.Time().Before(time.Unix(0, 0)) {
		t.Errorf("submitDate invalid")
	}
}

func TestTrainingRawRequest_Unmarshal_second_entry(t *testing.T) {
	js := `{
  "slug": "submit/252725624662359",
  "jsExecutionTracker": "build-date-1767092416391=>init-started:1767093629905=>validator-called:1767093629928=>validator-mounted-false:1767093629928=>init-complete:1767093629930=>interval-complete:1767093650929=>onsubmit-fired:1767093681178=>observerSubmitHandler_received-submit-event:1767093681179=>submit-validation-passed:1767093681193=>observerSubmitHandler_validation-passed-submitting-form:1767093681204",
  "submitSource": "form",
  "submitDate": "1767093681204",
  "buildDate": "1767092416391",
  "uploadServerUrl": "https://upload.jotform.com/upload",
  "eventObserver": "1",
  "q15_brcMembership15": "11111111",
  "q28_typeA28": [
    "Current Club Membership"
  ],
  "q18_horseName18": "test1",
  "q5_selectAltTestSession": {
    "implementation": "new",
    "date": "2026-01-03 18:00",
    "duration": "60",
    "timezone": "Europe/London (GMT+01:00)"
  },
  "q5_selectWestWiltsSession": {
    "implementation": "new",
    "date": "2026-01-01 18:00",
    "duration": "60",
    "timezone": "Europe/London (GMT+01:00)"
  },
  "q34_selectedVenue": "WestWilts",
  "q31_amount": "21",
  "q58_totalAmount": "37",
  "q53_paymentRef": "VSHE",
  "q12_typeA": "VSHE",
  "q54_wwecnonmem": "26",
  "q55_wwecmem": "21",
  "q56_widnonmem": "20",
  "q57_widmem": "16",
  "q48_brcMembership15-2": "22222222",
  "q49_typeA28-2": [
    "Current Club Membership"
  ],
  "q50_horseName18-2": "test2",
  "q51_selectAltTestVenueSession-2": {
    "implementation": "new",
    "date": "",
    "duration": "60",
    "timezone": "Europe/London (GMT+00:00)"
  },
  "q51_selectWidbrookSession-2": {
    "implementation": "new",
    "date": "2026-01-02 18:00",
    "duration": "60",
    "timezone": "Europe/London (GMT+00:00)"
  },
  "q60_selectedVenue-2": "Widbrook",
  "q59_amount-2": "16",
  "timeToSubmit": "20",
  "preview": "true",
  "validatedNewRequiredFieldIDs": "{\"new\":1,\"input_12\":\"VS\",\"id_15\":\"11\",\"id_18\":\"te\",\"id_5\":\"20\",\"id_48\":\"22\",\"id_50\":\"te\",\"id_51\":\"20\"}",
  "visitedPages": "{\"1\":true,\"2\":true}",
  "path": "/submit/252725624662359"
}`
	var rr TrainingRawRequest
	if err := json.Unmarshal([]byte(js), &rr); err != nil {
		t.Fatal(err)
	}

	_, _, startday := rr.Entries[0].SelectSession.StartLocal.Date()
	if startday != 1 {
		t.Errorf("date parse failed")
	}

	_, _, startday = rr.Entries[1].SelectSession.StartLocal.Date()
	if startday != 2 {
		t.Errorf("date parse failed")
	}

	if rr.Entries[0].Venue != "WestWilts" {
		t.Errorf("venue parse failed")
	}

	if rr.Entries[1].Venue != "Widbrook" {
		t.Errorf("venue parse failed")
	}

	if rr.SubmitDate.Time().Before(time.Unix(0, 0)) {
		t.Errorf("submitDate invalid")
	}
}
