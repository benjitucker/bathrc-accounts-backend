package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
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

	// defaultPaymentWindow is how long a member has to pay after being sent a payment link.
	defaultPaymentWindow = 2 * time.Hour
)

var (
	// Controlled by the PAYMENT_MODE env var
	paymentMode = paymentModeBACS

	// Controlled by the TRAINING_FORM_ID env var. The hourly catch-up fetches recent
	// submissions of this form, and the redirect only accepts submissions of this form.
	trainingFormID = productionTrainingFormID

	// Controlled by the PAYMENT_WINDOW env var (Stripe mode)
	paymentWindow = defaultPaymentWindow
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

// parsePaymentWindow validates the PAYMENT_WINDOW setting, e.g. "2h" or "15m". An empty value
// gives the default of 2 hours.
func parsePaymentWindow(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultPaymentWindow, nil
	}
	d, err := time.ParseDuration(value)
	if err != nil || d < time.Minute {
		return 0, fmt.Errorf("invalid PAYMENT_WINDOW %q: must be a duration of at least 1m, such as 2h", value)
	}
	return d, nil
}
