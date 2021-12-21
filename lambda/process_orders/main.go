package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	awsEvents "github.com/aws/aws-lambda-go/events"
	awsLambda "github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/glue"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	krakenapi "github.com/beldur/kraken-go-api-client"
	dcaConfig "github.com/kiran94/dca-manager/configuration"
	"github.com/kiran94/dca-manager/orders"
	log "github.com/sirupsen/logrus"
)

var (
	exchangeAttribute string = "Exchange"
	realAttribute     string = "Real"
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetReportCaller(false)
	log.Info("Lambda Execution Starting")

	if os.Getenv("_LAMBDA_SERVER_PORT") != "" {

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

func HandleRequest(c context.Context, event awsEvents.SQSEvent) (*string, error) {
	awsConfig, err := awsConfig.LoadDefaultConfig(c)
	if err != nil {
		return nil, err
	}

	err = ProcessTransactions(&awsConfig, &c, event)
	return nil, err
}

// Process Pending Transactions in the Queue.
// Pulls from the Queue and gets the details from the downstream Exchange.
// Pushes the details to S3 and marks as done from the Queue.
func ProcessTransactions(awsConfig *aws.Config, context *context.Context, sqsEvent awsEvents.SQSEvent) error {
	log.Info("Getting Transaction Details")

	// Call Kraken API
	key, secret, ssmErr := dcaConfig.GetKrakenDetails(*awsConfig, *context)
	if ssmErr != nil {
		return ssmErr
	}

	// Create Orderer (per Exchange)
	o := map[string]orders.Orderer{}
	o["kraken"] = orders.KrakenOrderer{
		Client: krakenapi.New(*key, *secret),
	}

	s3Client := s3.NewFromConfig(*awsConfig)
	sqsClient := sqs.NewFromConfig(*awsConfig)
	glueClient := glue.NewFromConfig(*awsConfig)

	if len(sqsEvent.Records) == 0 {
		log.Warn("No SQS Messages found, returning")
		return nil
	}

	// Process Each of the SQS Messages
	for _, message := range sqsEvent.Records {
		log.Infof("Processing SQS Message: %s from %s", message.MessageId, message.EventSourceARN)

		// Extract Details from the Message
		exchange := message.MessageAttributes[exchangeAttribute]
		realAtt := message.MessageAttributes[realAttribute]

		// If the message is a fake/testing message, mark as deleted and continue
		if *realAtt.StringValue == "false" {
			log.Warnf("Recieved SQS message which was not real. Deleting MessageId %s", message.MessageId)
			sqsClient.DeleteMessage(*context, &sqs.DeleteMessageInput{
				QueueUrl:      &message.EventSourceARN,
				ReceiptHandle: &message.ReceiptHandle,
			})

			continue
		}

		if exchange.StringValue == nil {
			log.Warnf("Recieved SQS message with no Exchange Set. Skipping MessageId: %s", message.MessageId)
			continue
		}

		messageBytes := []byte(message.Body)

		var po orders.PendingOrders
		err := json.Unmarshal(messageBytes, &po)
		if err != nil {
			log.Error("Unable to unmarshal json from Message %s", message.MessageId)
			return err
		}

		// Process the Transaction
		log.Infof("Processing Exchange: %s, Transactions: %s", *exchange.StringValue, po.TransactionId)
		orders, err := o[*exchange.StringValue].ProcessTransaction(po.TransactionId)
		if err != nil {
			return err
		}

		log.Debugf("Orders from processed transactions: %s", orders)

		// Upload Details to S3
		s3Bucket := os.Getenv(dcaConfig.EnvS3Bucket)
		s3PathPrefix := os.Getenv(dcaConfig.EnvS3ProcessedTransaction)

		for _, order := range *orders {

			if order.TransactionId == "" {
				log.Warnf("Found an order with no transaction id: %s", order)
				continue
			}

			s3Path := fmt.Sprintf(
				"%s/exchange=%s/%s.json",
				s3PathPrefix,
				strings.ToLower(*exchange.StringValue),
				order.TransactionId,
			)

			orderBytes, err := json.Marshal(order)
			if err != nil {
				return err
			}

			log.Infof("Uploading Transaction %s to Bucket %s, Key %s", order.TransactionId, s3Bucket, s3Path)
			_, s3PutErr := s3Client.PutObject(*context, &s3.PutObjectInput{
				Bucket: &s3Bucket,
				Key:    &s3Path,
				Body:   bytes.NewReader(orderBytes),
			})

			if s3PutErr != nil {
				return err
			}

			// Submit Glue Job
			jobName := os.Getenv(dcaConfig.EnvGlueProcessTransactionJob)
			jobArguments := map[string]string{
				"--input_path":      fmt.Sprintf("s3a://%s/%s", s3Bucket, s3Path),
				"--write_operation": os.Getenv(dcaConfig.EnvGlueProcessTransactionOperation),
			}

			// TODO: Schedule Hudi Load
			// TODO: Need to check if the job exists
			// in case enable_analytics = false

			log.Infof("Submitting Transaction %s to Glue Job %s with Arguments %s", order.TransactionId, jobName, jobArguments)
			submittedJob, glueStartErr := glueClient.StartJobRun(*context, &glue.StartJobRunInput{
				JobName:   &jobName,
				Arguments: jobArguments,
			})

			if glueStartErr != nil {
				return glueStartErr
			}

			log.Infof("Transaction %s submitted under Glue Job: %s", order.TransactionId, *submittedJob.JobRunId)
		}

		// Delete from Queue
		log.Infof("Deleting MessageId %s", message.MessageId)
		sqsClient.DeleteMessage(*context, &sqs.DeleteMessageInput{
			QueueUrl:      &message.EventSourceARN,
			ReceiptHandle: &message.ReceiptHandle,
		})
	}

	return nil
}

func HandleRequestLocally() {
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

	res, err := HandleRequest(context.TODO(), event)
	if res != nil {
		fmt.Printf("Result: %s \n", *res)
	}

	if err != nil {
		fmt.Printf("Error: %s \n", err)
	}
}
