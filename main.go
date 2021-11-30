package main

import (
	"context"
	"fmt"
	"os"
	"time"

	awsEvents "github.com/aws/aws-lambda-go/events"
	awsLambda "github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/kiran94/dca-manager/lambda"
	log "github.com/sirupsen/logrus"
)

const (
	envLambdaServerPort = "_LAMBDA_SERVER_PORT"
	envOperation        = "DCA_OPERATION"
)

var (
	operation = os.Getenv(envOperation)
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetReportCaller(false)
	log.Info("Lambda Execution Starting")

	lambdaPort := os.Getenv(envLambdaServerPort)

	if lambdaPort != "" {
		// Running in Lambda
		log.SetFormatter(&log.TextFormatter{
			DisableColors: true,
			FullTimestamp: true,
		})
		log.SetFormatter(&log.JSONFormatter{})

		awsLambda.Start(HandleRequest)

	} else {
		HandleRequestLocally()
	}

	log.Info("Lambda Execution Done.")
}

func HandleRequest(c context.Context, event interface{}) (*string, error) {
	log.Info("Starting Handling Request")

	awsConfig, err := awsConfig.LoadDefaultConfig(c)
	if err != nil {
		return nil, err
	}

	if operation == "EXECUTE_ORDERS" {
		err = lambda.ExecuteOrders(&awsConfig, &c)
	} else if operation == "PROCESS_TRANSACTIONS" {
		log.Infof("Recieved Event %s", event)
		sqsEvent := event.(awsEvents.SQSEvent)
		err = lambda.ProcessTransactions(&awsConfig, &c, sqsEvent)
	} else {
		err = fmt.Errorf("Unconfigured operation: %s", operation)
	}

	if err != nil {
		return nil, err
	}

	res := fmt.Sprintf("Operation %s completed successfully.", operation)
	return &res, nil
}

// Running the Lambda Locally.
func HandleRequestLocally() {

	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})

	var res *string
	var err error

	if operation == "EXECUTE_ORDERS" {

		event := awsEvents.CloudWatchEvent{
			Version:    "",
			ID:         "",
			DetailType: "",
			Source:     "",
			AccountID:  "",
			Time:       time.Time{},
			Region:     "",
			Resources:  []string{},
			Detail:     []byte{},
		}

		res, err = HandleRequest(context.TODO(), event)

	} else if operation == "PROCESS_TRANSACTIONS" {

		event := awsEvents.SQSEvent{
			Records: []awsEvents.SQSMessage{
				{
					MessageId:              "9dd0b57-b21e-4ac1-bd88-01bbb068cb78",
					ReceiptHandle:          "MessageReceiptHandle",
					Body:                   "",
					Md5OfBody:              "",
					Md5OfMessageAttributes: "",
					Attributes:             map[string]string{},
					MessageAttributes: map[string]awsEvents.SQSMessageAttribute{
						"Exchange": {
							DataType:    *aws.String("String"),
							StringValue: aws.String("exchange"),
						},
						"Real": {
							DataType:    *aws.String("String"),
							StringValue: aws.String("false"),
						},
					},
					EventSourceARN: "",
					EventSource:    "",
					AWSRegion:      "",
				},
			},
		}

		res, err = HandleRequest(context.TODO(), event)
	} else {
		err = fmt.Errorf("An operation must be set via %s", envOperation)
	}

	if res != nil {
		fmt.Printf("Result: %s \n", *res)
	} else {
		fmt.Println("Result was nil")
	}

	if err != nil {
		fmt.Printf("Error: %s \n", err)
	} else {
		log.Debug("Error was null")
	}
}
