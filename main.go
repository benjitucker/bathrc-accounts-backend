package main

import (
	"benjitucker/bathrc-accounts/db"
	"benjitucker/bathrc-accounts/email"
	"benjitucker/bathrc-accounts/jotform"
	"benjitucker/bathrc-accounts/jotform-webhook"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

const (
	trainingRequestForm = "Training"
	trainingAdminForm   = "Training Administration"
)

var (
	ctx                      context.Context
	trainTable               db.TrainingSubmissionTable
	memberTable              db.MemberTable
	transactionTable         db.TransactionTable
	jotformClient            *jotform.APIClient
	emailHandler             *email.EmailHandler
	ssmClient                *ssm.Client
	clubEmail, trainingEmail string
	testEmail, testEmail2    string

	// Controlled by the TEST_MODE env var
	testMode = false
)

type EventBridgePayload struct {
	PeriodType string `json:"period"`
}

func handler(raw json.RawMessage) (any, error) {

	// Try API Gateway first
	var apiReq events.LambdaFunctionURLRequest
	if err := json.Unmarshal(raw, &apiReq); err == nil && apiReq.Body != "" {
		return handleAPIRequest(apiReq)
	}

	// Try EventBridge / CloudWatch Event with our custom request payload
	var eb EventBridgePayload
	if err := json.Unmarshal(raw, &eb); err == nil && eb.PeriodType != "" {
		return handleEventBridge(eb)
	}

	// Fallback
	fmt.Println("Unknown event type")
	return map[string]string{"status": "unhandled"}, nil
}

func handleEventBridge(payload EventBridgePayload) (any, error) {
	fmt.Printf("Handle Event Bridge, period %s\n", payload.PeriodType)

	if payload.PeriodType == "hourly" {
		err := handleHourly(false)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			emailHandler.SendEmail(testEmail, "jotform event bridge: FAIL", err.Error())
			return nil, err
		}
	}

	if payload.PeriodType == "run-test" {
		err := handleHourly(true)
		if err != nil {
			return nil, err
		}
	}

	return map[string]string{
		"message": "ok",
	}, nil
}

func handleAPIRequest(req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	fmt.Printf("Handle API, body %s", req.Body)

	formData, err := jotform_webhook.DecodeBase64Multipart(req.Body)
	if err != nil {
		return serverError(err)
	}

	fmt.Printf("%s", formData.DebugString())

	switch formData.RawRequest.FormKind() {
	case trainingRequestForm:
		request := formData.RawRequest.(jotform_webhook.TrainingRawRequest)
		err = handleTrainingRequest(formData.SubmissionID, &request)
	case trainingAdminForm:
		err = handleTrainingAdmin(formData, formData.RawRequest.(jotform_webhook.TrainingAdminRawRequest))
	default:
		err = fmt.Errorf("unknown form kind: %s", formData.RawRequest.FormKind())
	}

	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		emailHandler.SendEmail(testEmail, "jotform webhook: FAIL", err.Error())
	}

	resp := events.LambdaFunctionURLResponse{
		StatusCode:      200,
		IsBase64Encoded: false,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	return resp, nil
}

func serverError(err error) (events.LambdaFunctionURLResponse, error) {
	fmt.Printf("ERROR: %v\n", err)

	return events.LambdaFunctionURLResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       http.StatusText(http.StatusInternalServerError),
	}, nil
}

func main() {
	ctx = context.Background()

	flag.Parse()

	logLevel, exists := os.LookupEnv("LOG_LEVEL")
	if !exists {
		logLevel = "debug"
	}

	tm, exists := os.LookupEnv("TEST_MODE")
	if exists && strings.ToLower(tm) != "false" {
		testMode = true
		fmt.Printf("TEST MODE ENABLED!\n")
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}

	ddb := dynamodb.NewFromConfig(cfg)
	sesClient := ses.NewFromConfig(cfg)
	ssmClient = ssm.NewFromConfig(cfg)
	clubEmail = getSecret("club-email-address")
	trainingEmail = getSecret("training-email-address")
	testEmail = getSecret("test-email-address")
	testEmail2 = getSecret("test-email-address2")

	emailHandler, err = email.NewEmailHandler(ctx, sesClient, email.HandlerParams{
		AccountNumber: getSecret("bathrc-account-number"),
		SortCode:      getSecret("bathrc-sort-code"),
		MonitorEmail:  testEmail,
		ClubEmail:     clubEmail,
		TrainingEmail: trainingEmail,
	})
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}

	// TODO - do not open everything if you dont need to

	err = trainTable.Open(ctx, ddb)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}

	err = memberTable.Open(ctx, ddb)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}

	err = transactionTable.Open(ctx, ddb)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}

	jotformClient = jotform.NewJotFormAPIClient(
		getSecret("bathrc-jotform-apikey"), "json", logLevel == "debug")

	lambda.Start(handler)
}

func getSecret(paramName string) string {
	withDecryption := true
	resp, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           &paramName,
		WithDecryption: &withDecryption,
	})
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return ""
	}
	return *resp.Parameter.Value
}
