package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"benjitucker/bathrc-accounts/jotform"
)

func main() {
	apiKey := flag.String("api-key", os.Getenv("JOTFORM_API_KEY"), "JotForm API Key (can also be set via JOTFORM_API_KEY env var)")
	formIDStr := flag.String("form-id", "", "Specific Form ID to dump submissions for (optional)")
	limit := flag.Int("limit", 100, "Number of results per request (max 1000)")
	offset := flag.Int("offset", 0, "Start of each result set")
	all := flag.Bool("all", false, "Dump all submissions (handles pagination automatically)")
	since := flag.Duration("since", 0, "Only dump submissions since this duration ago (e.g. 336h for 2 weeks)")

	flag.Parse()

	if *apiKey == "" {
		log.Fatal("API Key is required. Use -api-key flag or set JOTFORM_API_KEY environment variable.")
	}

	client := jotform.NewJotFormAPIClient(*apiKey, "json", false)

	var filter map[string]string
	if *since > 0 {
		cutoff := time.Now().Add(-*since).Format("2006-01-02 15:04:05")
		filter = map[string]string{
			"created_at:gt": cutoff,
		}
	}

	if *all {
		dumpAllSubmissions(client, *formIDStr, *limit, filter)
	} else {
		dumpSubmissions(client, *formIDStr, *offset, *limit, filter)
	}
}

func dumpSubmissions(client *jotform.APIClient, formIDStr string, offset, limit int, filter map[string]string) {
	var resp []byte
	var err error

	offsetStr := strconv.Itoa(offset)
	limitStr := strconv.Itoa(limit)

	if formIDStr != "" {
		formID, err := strconv.ParseInt(formIDStr, 10, 64)
		if err != nil {
			log.Fatalf("Invalid form ID: %v", err)
		}
		resp, err = client.GetFormSubmissions(formID, offsetStr, limitStr, filter, "")
	} else {
		resp, err = client.GetSubmissions(offsetStr, limitStr, filter, "")
	}

	if err != nil {
		log.Fatalf("Error fetching submissions: %v", err)
	}

	var result struct {
		Content []interface{} `json:"content"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		// Fallback to raw output if unmarshal fails
		fmt.Println(string(resp))
		return
	}

	for _, sub := range result.Content {
		subJSON, _ := json.Marshal(sub)
		fmt.Println(string(subJSON))
	}
}

func dumpAllSubmissions(client *jotform.APIClient, formIDStr string, limit int, filter map[string]string) {
	offset := 0
	for {
		var resp []byte
		var err error

		offsetStr := strconv.Itoa(offset)
		limitStr := strconv.Itoa(limit)

		if formIDStr != "" {
			formID, _ := strconv.ParseInt(formIDStr, 10, 64)
			resp, err = client.GetFormSubmissions(formID, offsetStr, limitStr, filter, "")
		} else {
			resp, err = client.GetSubmissions(offsetStr, limitStr, filter, "")
		}

		if err != nil {
			log.Fatalf("Error fetching submissions at offset %d: %v", offset, err)
		}

		var result struct {
			Content []interface{} `json:"content"`
		}
		if err := json.Unmarshal(resp, &result); err != nil {
			// If we can't unmarshal, just print the raw response and stop
			fmt.Println(string(resp))
			return
		}

		if len(result.Content) == 0 {
			break
		}

		for _, sub := range result.Content {
			subJSON, _ := json.Marshal(sub)
			fmt.Println(string(subJSON))
		}

		offset += len(result.Content)
		if len(result.Content) < limit {
			break
		}
	}
}
