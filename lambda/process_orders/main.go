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
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/glue"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/kiran94/dca-manager/pkg"
	"github.com/kiran94/dca-manager/pkg/configuration"
	"github.com/kiran94/dca-manager/pkg/orders"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

var (
	dcaServices *DCAServices
	appConfig   *AppConfig
)

type DCAServices struct {
	awsConfig             aws.Config
	s3Access              pkg.S3Access
	ssmAccess             pkg.SSMAccess
	sqsAccess             pkg.SQSAccess
	glueAccess            pkg.GlueAccess
	configSource          configuration.DCAConfigurationSource
	ordererFactory        orders.OrdererFactory
	pendingOrderSubmitter orders.PendingOrderQueue
}

type AppConfig struct {
	s3bucket      string
	dcaConfigPath string
	allowReal     bool
	transactions  struct {
		pendingS3TransactionPrefix   string
		processedS3TransactionPrefix string
	}
	queue struct {
		sqsUrl string
	}
	glue struct {
		processTransactionJob       string
		processTransactionOperation string
	}
}

func init() {
	awsConfig, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		logrus.WithError(err).Panic("Could not retrieve default aws config")
	}

	dcaServices = &DCAServices{}
	dcaServices.awsConfig = awsConfig
	dcaServices.s3Access = pkg.S3{Client: s3.NewFromConfig(awsConfig)}
	dcaServices.ssmAccess = pkg.SSM{Client: ssm.NewFromConfig(awsConfig)}
	dcaServices.sqsAccess = pkg.SQS{Client: sqs.NewFromConfig(awsConfig)}
	dcaServices.glueAccess = pkg.Glue{Client: glue.NewFromConfig(awsConfig)}
	dcaServices.ordererFactory = orders.OrdererFac{}
	dcaServices.configSource = configuration.DCAConfiguration{}
	dcaServices.pendingOrderSubmitter = orders.PendingOrderSubmitter{}

	appConfig = &AppConfig{
		s3bucket:      os.Getenv(configuration.EnvS3Bucket),
		dcaConfigPath: os.Getenv(configuration.EnvS3ConfigPath),
		allowReal:     os.Getenv(configuration.EnvAllowReal) != "",
	}
	appConfig.transactions.pendingS3TransactionPrefix = os.Getenv(configuration.EnvS3PendingTransaction)
	appConfig.transactions.processedS3TransactionPrefix = os.Getenv(configuration.EnvS3ProcessedTransaction)
	appConfig.queue.sqsUrl = os.Getenv(configuration.EnvSQSPendingOrdersQueue)
	appConfig.glue.processTransactionJob = os.Getenv(configuration.EnvGlueProcessTransactionJob)
	appConfig.glue.processTransactionOperation = os.Getenv(configuration.EnvGlueProcessTransactionOperation)
}

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

func HandleRequest(ctx context.Context, event awsEvents.SQSEvent) (*string, error) {
	if err := ProcessTransactions(ctx, dcaServices, appConfig, event); err != nil {
		return nil, err
	}

	return nil, nil
}

// Process Pending Transactions in the Queue.
// Pulls from the Queue and gets the details from the downstream Exchange.
// Pushes the details to S3 and marks as done from the Queue.
func ProcessTransactions(ctx context.Context, dcaServices *DCAServices, appConfig *AppConfig, sqsEvent awsEvents.SQSEvent) error {
	log.Info("Getting Transaction Details")

	if len(sqsEvent.Records) == 0 {
		return fmt.Errorf("No SQS Messages found, returning")
	}

	o, err := dcaServices.ordererFactory.GetOrderers(ctx, dcaServices.ssmAccess)
	if err != nil {
		return err
	}

	// Process Each of the SQS Messages
	for _, message := range sqsEvent.Records {
		log.Infof("Processing SQS Message: %s from %s", message.MessageId, message.EventSourceARN)

		// Extract Details from the Message
		exchange := message.MessageAttributes["Exchange"]
		realAtt := message.MessageAttributes["Real"]

		// If the message is a fake/testing message, mark as deleted and continue
		if *realAtt.StringValue == "false" {
			log.Warnf("Recieved SQS message which was not real. Deleting MessageId %s from queue %s", message.MessageId, message.EventSourceARN)
			_, err = dcaServices.sqsAccess.DeleteMessage(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      &message.EventSourceARN,
				ReceiptHandle: &message.ReceiptHandle,
			})

			if err != nil {
				return err
			}

			continue
		}

		if exchange.StringValue == nil || *exchange.StringValue == "" {
			return fmt.Errorf("recieved sqs message with no exchange set. Skipping message %s", message.MessageId)
		}

		messageBytes := []byte(message.Body)

		var po orders.PendingOrders
		err := json.Unmarshal(messageBytes, &po)
		if err != nil {
			log.Errorf("Unable to unmarshal json from Message %s", message.MessageId)
			return err
		}

		// Process the Transaction
		log.Infof("Processing Exchange: %s, Transactions: %s", *exchange.StringValue, po.TransactionId)
		exchangeOrderer, ok := (*o)[*exchange.StringValue]
		if !ok {
			return fmt.Errorf("exchange %s was not configured", *exchange.StringValue)
		}

		orders, err := exchangeOrderer.ProcessTransaction(po.TransactionId)
		if err != nil {
			return err
		}

		log.Debugf("Orders from processed transactions: %v", orders)

		// Upload Details to S3
		s3Bucket := appConfig.s3bucket
		s3PathPrefix := appConfig.dcaConfigPath

		for _, order := range *orders {

			if order.TransactionId == "" {
				log.Warnf("Found an order with no transaction id: %v", order)
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
			_, s3PutErr := dcaServices.s3Access.PutObject(ctx, &s3.PutObjectInput{
				Bucket: &s3Bucket,
				Key:    &s3Path,
				Body:   bytes.NewReader(orderBytes),
			})

			if s3PutErr != nil {
				return s3PutErr
			}

			// Since we are passing the absolute complete path for the loaded JSON file
			// the spark won't be able to derive any hive partition columns
			// so here we are adding the exchange as an additional column
			additional_columns := map[string]string{"exchange": strings.ToLower(*exchange.StringValue)}
			additional_columns_json, addErr := json.Marshal(additional_columns)
			if addErr != nil {
				return addErr
			}

			// Submit Glue Job
			jobName := appConfig.glue.processTransactionJob
			jobArguments := map[string]string{
				"--input_path":         fmt.Sprintf("s3a://%s/%s", s3Bucket, s3Path),
				"--write_operation":    appConfig.glue.processTransactionOperation,
				"--additional_columns": string(additional_columns_json),
			}

			log.Infof("Submitting Transaction %s to Glue Job %s with Arguments %s", order.TransactionId, jobName, jobArguments)
			submittedJob, glueStartErr := dcaServices.glueAccess.StartJobRun(ctx, &glue.StartJobRunInput{
				JobName:   &jobName,
				Arguments: jobArguments,
			})

			if glueStartErr != nil {
				return glueStartErr
			}

			log.Infof("Transaction %s submitted under Glue Job: %s", order.TransactionId, *submittedJob.JobRunId)
		}

		// Delete from Queue
		log.Infof("Deleting MessageId %s from queue %s", message.MessageId, message.EventSourceARN)
		dcaServices.sqsAccess.DeleteMessage(ctx, &sqs.DeleteMessageInput{
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
				EventSourceARN: "fake_eventsourcearn",
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
