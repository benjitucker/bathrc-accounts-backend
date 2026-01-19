package main

import (
	"benjitucker/bathrc-accounts/db"
	jotform_webhook "benjitucker/bathrc-accounts/jotform-webhook"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	// TODO hard coded
	formId          = 252725624662359
	jotformLocation = "America/New_York"
)

// handleSubmissionsCheck fetches recent Jotform submissions and compares them against the provided list of database submissions.
// It identifies any submissions missing from the database and processes them using handleTrainingRequest.
func handleSubmissionsCheck(submissions []*db.TrainingSubmission) error {
	loc, err := time.LoadLocation(jotformLocation)
	if err != nil {
		fmt.Println("Error loading location:", err)
		return nil
	}

	// Get info on all submissions made in the last 4 hours
	submissionsData, err := jotformClient.GetFormSubmissions(formId, "0", "100", map[string]string{
		"created_at:gt": time.Now().In(loc).Add(-time.Hour * 4).Format("2006-01-02 15:04:05"),
	}, "created_at")
	if err != nil {
		return fmt.Errorf("failed getting form submissions: %w", err)
	}

	var content []jotform_webhook.TrainingRawRequestWithID

	if err := json.Unmarshal(submissionsData, &content); err != nil {
		return fmt.Errorf("failed to unmarshal api JSON %s: %w", string(submissionsData), err)
	}

	fmt.Printf("Successfully got %d submissions made in the last 4 hours from API\n", len(content))

	// Check for any submissions we do not have in our database
	for _, apiSubmission := range content {
		submissionId := apiSubmission.SubmissionID
		foundInDb := false
		for _, dbSubmission := range submissions {
			// Submission DB record's ID are formatted: "<submissionId>-<entryNum>"
			if strings.HasPrefix(dbSubmission.GetID(), submissionId) {
				foundInDb = true
				break
			}
		}
		if foundInDb == true {
			continue
		}

		fmt.Printf("Submission id: %s was not found in db, need to process this submission\n",
			submissionId)

		err = handleTrainingRequest(submissionId, &apiSubmission)
		if err != nil {
			return err
		}
	}

	return nil
}
