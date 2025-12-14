package main

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
)

const (
	awsRegion = "eu-west-3" // AWS region where SES is configured
	sender    = "support@bathridingclub.co.uk"
)

func sendEmail(recipient, subject, body string) {

	// Sender and recipient
	/*
		sender := "support@bathridingclub.co.uk"
		recipient := "recipient@example.com"

		// Email subject and body
		subject := "Test Email from Bath Riding Club"
		body := "Hello,\n\nThis is a test email sent via AWS SES in Go.\n\nBest regards,\nBath Riding Club"

	*/

	// Create a new session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(awsRegion),
	})
	if err != nil {
		log.Fatalf("failed to create AWS session: %v", err)
	}

	// Create SES service client
	svc := ses.New(sess)

	// Build the email input
	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			ToAddresses: []*string{
				aws.String(recipient),
			},
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Text: &ses.Content{
					Charset: aws.String("UTF-8"),
					Data:    aws.String(body),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String("UTF-8"),
				Data:    aws.String(subject),
			},
		},
		Source: aws.String(sender),
	}

	// Send the email
	result, err := svc.SendEmail(input)
	if err != nil {
		log.Fatalf("failed to send email: %v", err)
	}

	fmt.Println("Email sent! Message ID:", *result.MessageId)
}
