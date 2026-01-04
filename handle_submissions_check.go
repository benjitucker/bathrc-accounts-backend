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
	formId = 252725624662359
)

func handleSubmissionsCheck(submissions []*db.TrainingSubmission) error {
	// Get info on all submissions made in the last 90 minutes
	submissionsData, err := jotformClient.GetFormSubmissions(formId, "0", "100", map[string]string{
		"created_at:gt": time.Now().Add(-time.Minute * 90).Format("2006-01-02 15:04:05"),
	}, "created_at")
	if err != nil {
		return fmt.Errorf("failed getting form submissions: %w", err)
	}

	var env jotform_webhook.APIEnvelope

	if err := json.Unmarshal(submissionsData, &env); err != nil {
		return fmt.Errorf("failed to unmarshal api JSON %s: %w", string(submissionsData), err)
	}

	if env.ResponseCode != 200 || env.Message != "success" {
		fmt.Printf("jotform get form submissions failed: %s", string(submissionsData))
		return nil
	}

	fmt.Printf("Successfully got %d submissions made in the last 90 minutes from API\n", len(env.Content))

	// Check for any submissions we do not have in our database
	for _, apiSubmission := range env.Content {
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

		fmt.Printf("Submission id: %s was not found in db, need to process this submission",
			submissionId)

		err = handleTrainingRequest(submissionId, &apiSubmission)
		if err != nil {
			return err
		}
	}

	return nil
}
