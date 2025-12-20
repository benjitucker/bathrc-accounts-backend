package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

const (
	sender = "support@bathridingclub.co.uk"
)

func sendEmail(ctx context.Context, recipient, subject, body string) {

	// Build the email input
	input := &ses.SendEmailInput{
		Destination: &types.Destination{
			ToAddresses: []string{recipient},
		},
		Message: &types.Message{
			Subject: &types.Content{
				Charset: aws.String("UTF-8"),
				Data:    aws.String(subject),
			},
			Body: &types.Body{
				Text: &types.Content{
					Charset: aws.String("UTF-8"),
					Data:    aws.String(body),
				},
			},
		},
		Source: aws.String(sender),
	}

	// Send the email
	result, err := sesClient.SendEmail(ctx, input)
	if err != nil {
		log.Fatalf("failed to send email: %v", err)
	}

	fmt.Println("Email sent! Message ID:", *result.MessageId)
}
