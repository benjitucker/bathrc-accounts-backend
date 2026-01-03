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

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

const (
	trainingRequestForm = "Training"
	trainingAdminForm   = "Training Administration"
)

var (
	ctx                      context.Context
	logger                   log.Logger
	trainTable               db.TrainingSubmissionTable
	memberTable              db.MemberTable
	transactionTable         db.TransactionTable
	jotformClient            *jotform.APIClient
	emailHandler             *email.EmailHandler
	ssmClient                *ssm.Client
	clubEmail, trainingEmail string
	testEmail, testEmail2    string
	testMode                 = true // TODO - disable
)

type EventBridgePayload struct {
	PeriodType string `json:"period"`
}

func handler(raw json.RawMessage) (any, error) {

	input, _ := raw.MarshalJSON() // TODO remove
	fmt.Println(string(input))    // TODO remove

	// Try EventBridge / CloudWatch Event
	var eb events.CloudWatchEvent
	if err := json.Unmarshal(raw, &eb); err == nil && eb.Source != "" {
		return handleEventBridge(eb)
	}

	// Try API Gateway first
	var apiReq events.APIGatewayProxyRequest
	if err := json.Unmarshal(raw, &apiReq); err == nil && apiReq.HTTPMethod != "" {
		return handleAPIRequest(apiReq)
	}

	// Fallback
	fmt.Println("Unknown event type")
	return map[string]string{"status": "unhandled"}, nil
}

func handleEventBridge(eb events.CloudWatchEvent) (any, error) {
	var payload EventBridgePayload
	if err := json.Unmarshal(eb.Detail, &payload); err != nil {
		return nil, err
	}

	if payload.PeriodType == "hourly" {
		err := handleHourly(false)
		if err != nil {
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

func handleAPIRequest(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger := log.With(logger, "method", "HandleRequest")
	_ = level.Debug(logger).Log("msg", "Handle Request", "body", req.Body)

	formData, err := jotform_webhook.DecodeBase64Multipart(req.Body)
	if err != nil {
		return serverError(err)
	}

	_ = level.Debug(logger).Log("msg", "Handle Request", "form", formData.DebugString())

	switch formData.RawRequest.FormKind() {
	case trainingRequestForm:
		err = handleTrainingRequest(formData, formData.RawRequest.(jotform_webhook.TrainingRawRequest))
	case trainingAdminForm:
		err = handleTrainingAdmin(formData, formData.RawRequest.(jotform_webhook.TrainingAdminRawRequest))
	default:
		err = fmt.Errorf("unknown form kind: %s", formData.RawRequest.FormKind())
	}

	if err != nil {
		emailHandler.SendEmail(testEmail, "jotform webhook: FAIL", err.Error())
	}

	resp := events.APIGatewayProxyResponse{
		StatusCode:      200,
		IsBase64Encoded: false,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	return resp, nil
}

func serverError(err error) (events.APIGatewayProxyResponse, error) {
	logger := log.With(logger, "method", "serverError")
	_ = level.Error(logger).Log("err", err)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       http.StatusText(http.StatusInternalServerError),
	}, nil
}

func main() {
	ctx = context.Background()
	logger = log.NewLogfmtLogger(os.Stderr)
	logger = log.NewSyncLogger(logger)
	logger = log.With(logger,
		"service", "bathrc-accounts-backend",
		"time:", log.DefaultTimestampUTC,
		"caller", log.DefaultCaller,
	)

	_ = level.Info(logger).Log("msg", "service started")
	defer func() { _ = level.Info(logger).Log("msg", "service finished") }()

	flag.Parse()

	logLevel, exists := os.LookupEnv("LOG_LEVEL")
	if !exists {
		logLevel = "debug"
	}

	switch logLevel {
	case "debug":
		logger = level.NewFilter(logger, level.AllowDebug())
	case "info":
		logger = level.NewFilter(logger, level.AllowInfo())
	case "warn":
		logger = level.NewFilter(logger, level.AllowWarn())
	case "error":
		logger = level.NewFilter(logger, level.AllowError())
	default:
		logger = level.NewFilter(logger, level.AllowAll())
		_ = level.Error(logger).Log("msg", "bad logging level, defaulting to all")
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		_ = level.Error(logger).Log("unable to load AWS config: %v", err)
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
		_ = level.Error(logger).Log("unable to open create new email handler: %v", err)
		return
	}

	// TODO - do not open everything if you dont need to

	err = trainTable.Open(ctx, ddb)
	if err != nil {
		_ = level.Error(logger).Log("unable to open training submissions table: %v", err)
		return
	}

	err = memberTable.Open(ctx, ddb)
	if err != nil {
		_ = level.Error(logger).Log("unable to open members table: %v", err)
		return
	}

	err = transactionTable.Open(ctx, ddb)
	if err != nil {
		_ = level.Error(logger).Log("unable to open transaction table: %v", err)
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
		_ = level.Error(logger).Log("failed to get parameter: %v", err)
		return ""
	}
	return *resp.Parameter.Value
}
