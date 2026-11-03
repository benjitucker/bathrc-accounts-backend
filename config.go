package main

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// paymentModeBACS is the existing behaviour: members pay by bank transfer and the
	// /redirect endpoint is disabled. It is the default, so production is unchanged
	// unless PAYMENT_MODE is set.
	paymentModeBACS = "bacs"
	// paymentModeStripe enables the post-submission /redirect endpoint.
	paymentModeStripe = "stripe"

	// productionTrainingFormID is the live Training form, used unless TRAINING_FORM_ID is set.
	productionTrainingFormID int64 = 252725624662359
)

var (
	// Controlled by the PAYMENT_MODE env var
	paymentMode = paymentModeBACS

	// Controlled by the TRAINING_FORM_ID env var. The hourly catch-up fetches recent
	// submissions of this form, and the redirect only accepts submissions of this form.
	trainingFormID = productionTrainingFormID
)

// parsePaymentMode validates the PAYMENT_MODE setting. An empty value gives the default.
func parsePaymentMode(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", paymentModeBACS:
		return paymentModeBACS, nil
	case paymentModeStripe:
		return paymentModeStripe, nil
	default:
		return "", fmt.Errorf("invalid PAYMENT_MODE %q: must be %q or %q", value, paymentModeBACS, paymentModeStripe)
	}
}

// parseTrainingFormID validates the TRAINING_FORM_ID setting. An empty value gives the
// production form.
func parseTrainingFormID(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return productionTrainingFormID, nil
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid TRAINING_FORM_ID %q: must be a Jotform form number", value)
	}
	return id, nil
}
