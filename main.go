package main

import (
	"benjitucker/bathrc-accounts/db"
	"context"
	"flag"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

var (
	logger     log.Logger
	trainTable db.TrainingSubmissionTable
)

func HandleRequest(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger := log.With(logger, "method", "HandleRequest")
	_ = level.Debug(logger).Log("msg", "Handle Request", "body", req.Body)

	sendEmail("ben@churchfarmmonktonfarleigh.co.uk", "jotform webhook body", req.Body)

	formData, err := parseBase64Multipart(req.Body)
	if err != nil {
		return serverError(err)
	}

	_ = level.Debug(logger).Log("msg", "Handle Request", "form", formData.DebugString())

	err = trainTable.Put(&db.TrainingSubmission{
		DBItem: db.DBItem{
			ID: formData.SubmissionID,
		},
		Date:             formData.RawRequest.SelectSession.Date,
		MembershipNumber: formData.RawRequest.MembershipNumber,
	})
	if err != nil {
		return serverError(err)
	}

	records, err := trainTable.GetAll()
	if err != nil {
		return serverError(err)
	}

	_ = level.Debug(logger).Log("msg", "Handle Request", "number of records", len(records))
	for _, record := range records {
		_ = level.Debug(logger).Log("msg", "Handle Request", "record from db", record)
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

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		_ = level.Error(logger).Log("unable to load AWS config: %v", err)
		return
	}

	ddb := dynamodb.NewFromConfig(cfg)

	err = trainTable.Open(context.Background(), ddb)
	if err != nil {
		_ = level.Error(logger).Log("unable to open training submissions: %v", err)
		return
	}

	lambda.Start(HandleRequest)
}
