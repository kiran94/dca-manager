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
	logrus.SetOutput(os.Stdout)
	logrus.SetReportCaller(false)
	logrus.Info("Lambda Execution Starting")

	if os.Getenv("_LAMBDA_SERVER_PORT") != "" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
		awsLambda.Start(HandleRequest)

	} else {
		logrus.SetFormatter(&logrus.TextFormatter{})
		HandleRequestLocally()
	}

	logrus.Info("Lambda Execution Done.")
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
	logrus.Info("Processing Transaction Details")

	if len(sqsEvent.Records) == 0 {
		return fmt.Errorf("no sqs messages found, returning")
	}

	o, err := dcaServices.ordererFactory.GetOrderers(ctx, dcaServices.ssmAccess)
	if err != nil {
		return err
	}

	// Process Each of the SQS Messages
	for _, message := range sqsEvent.Records {
		// Extract Details from the Message
		exchange := message.MessageAttributes["Exchange"]
		realAtt := message.MessageAttributes["Real"]

		logrus.WithFields(logrus.Fields{
			"messageId":      message.MessageId,
			"eventSourceArn": message.EventSourceARN,
			"exchange":       *exchange.StringValue,
			"real":           *realAtt.StringValue,
		}).Info("Processing SQS Message")

		// If the message is a fake/testing message, mark as deleted and continue
		if *realAtt.StringValue == "false" {
			logrus.WithFields(logrus.Fields{
				"messageId": message.MessageId,
				"queue":     message.EventSourceARN,
			}).Warn("Received SQS message which was not real. Deleting from Queue.")

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
			logrus.Errorf("Unable to unmarshal json from Message %s", message.MessageId)
			return err
		}

		// Process the Transaction
		logrus.WithFields(logrus.Fields{
			"exchange":      *exchange.StringValue,
			"transactionId": po.TransactionId,
		}).Info("Processing Transaction")

		exchangeOrderer, ok := (*o)[*exchange.StringValue]
		if !ok {
			return fmt.Errorf("exchange %s was not configured", *exchange.StringValue)
		}

		orders, err := exchangeOrderer.ProcessTransaction(po.TransactionId)
		if err != nil {
			return err
		}
		logrus.WithField("order", orders).Debug("Orders from processed transaction")

		// Upload Details to S3
		s3Bucket := appConfig.s3bucket
		s3PathPrefix := appConfig.transactions.processedS3TransactionPrefix

		for _, order := range *orders {

			if order.TransactionId == "" {
				logrus.Warnf("Found an order with no transaction id: %v", order)
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

			logrus.WithFields(logrus.Fields{
				"transactionId": order.TransactionId,
				"s3bucket":      s3Bucket,
				"s3path":        s3Path,
			}).Info("Uploading Transaction result to S3")

			_, err = dcaServices.s3Access.PutObject(ctx, &s3.PutObjectInput{
				Bucket: &s3Bucket,
				Key:    &s3Path,
				Body:   bytes.NewReader(orderBytes),
			})

			if err != nil {
				return err
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

			logrus.WithFields(logrus.Fields{
				"glueJobName":       jobName,
				"inputS3Bucket":     s3Bucket,
				"inputPath":         s3Path,
				"writeOperation":    jobArguments["--write_operation"],
				"additionalColumns": jobArguments["--additional_columns"],
			}).Info("Submitting Glue Job")

			submittedJob, glueStartErr := dcaServices.glueAccess.StartJobRun(ctx, &glue.StartJobRunInput{
				JobName:   &jobName,
				Arguments: jobArguments,
			})

			if glueStartErr != nil {
				return glueStartErr
			}

			logrus.WithFields(logrus.Fields{
				"transactionId": order.TransactionId,
				"glueJobId":     *submittedJob.JobRunId,
			}).Info("Glue Job Submitted")
		}

		// Delete from Queue
		logrus.WithFields(logrus.Fields{
			"messageId":      message.MessageId,
			"eventSourceArn": message.EventSourceARN,
		}).Info("Deleting Message from Queue")

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

	res, err := HandleRequest(context.Background(), event)
	if res != nil {
		logrus.WithField("result", *res).Info("request successful locally.")
	}

	if err != nil {
		logrus.WithError(err).Error("Error running request locally.")
	}
}
