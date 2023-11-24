package main

import (
	"benjitucker/bathrc-accounts/housing_list"
	"benjitucker/bathrc-accounts/todo"
	"bytes"
	"encoding/json"
	"flag"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"io"
	"net/http"
	"os"
	"time"
)

var (
	logger log.Logger
)

func HandleRequest(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger := log.With(logger, "method", "HandleRequest")
	_ = level.Debug(logger).Log("msg", "Handle Request", "body", req.Body)
	var buf bytes.Buffer

	if req.Path == "/backend/housing_location" {
		body, err := json.Marshal(housing_list.Get())
		if err != nil {
			return serverError(err)
		}
		json.HTMLEscape(&buf, body)
	} else if req.Path == "/backend/todo" {
		body, err := json.Marshal(todo.Get())
		if err != nil {
			return serverError(err)
		}
		json.HTMLEscape(&buf, body)
	} else if req.Body == "test" {
		body, err := json.Marshal(req)
		if err != nil {
			return serverError(err)
		}
		json.HTMLEscape(&buf, body)
	} else {

		client := http.Client{
			Timeout: 5 * time.Second,
		}

		//dsResp, err := client.Get("https://ifconfig.me/ip")
		dsResp, err := client.Get(req.Body)
		if err != nil {
			return serverError(err)
		}

		rspBody, err := io.ReadAll(dsResp.Body)
		if err != nil {
			return serverError(err)
		}

		body, err := json.Marshal(map[string]interface{}{
			"rspBody": string(rspBody),
		})
		if err != nil {
			return serverError(err)
		}
		json.HTMLEscape(&buf, body)
	}

	resp := events.APIGatewayProxyResponse{
		StatusCode:      200,
		IsBase64Encoded: false,
		Body:            buf.String(),
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

	_ = level.Info(logger).Log("msg", "service started 21:56") /* TODO remove timestamp */
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

	lambda.Start(HandleRequest)
}
