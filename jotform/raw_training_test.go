package jotform

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
		"q5_selectSession":{
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

	if rr.SelectSession.Duration != 45 {
		t.Errorf("duration parse failed")
	}

	if rr.SelectSession.Date.Year() != 2025 {
		t.Errorf("date parse failed")
	}

	if rr.SubmitDate.Time().Before(time.Unix(0, 0)) {
		t.Errorf("submitDate invalid")
	}
}
