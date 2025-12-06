package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	logger      log.Logger
	mongoClient mongo.Client
)

func HandleRequest(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger := log.With(logger, "method", "HandleRequest")
	_ = level.Debug(logger).Log("msg", "Handle Request", "body", req.Body)
	var buf bytes.Buffer

	if req.Path == "/jotform-hook" {
		body, err := json.Marshal("hookie")
		if err != nil {
			return serverError(err)
		}
		json.HTMLEscape(&buf, body)
	} else {
		return serverError(errors.New("not found"))
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

	/*
		uri, exists := os.LookupEnv("MONGO_URI")
		if !exists {
			_ = level.Error(logger).Log("You must set your 'MONGO_URI' environment variable")
			panic(nil)
		}
		mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
		if err != nil {
			panic(err)
		}
		defer func() {
			if err := mongoClient.Disconnect(context.Background()); err != nil {
				panic(err)
			}
		}()

		housing_list.SetMongoClient(mongoClient)

	*/

	lambda.Start(HandleRequest)
}
