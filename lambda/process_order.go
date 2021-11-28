package lambda

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
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

// Process Pending Transactions in the Queue.
// Pulls from the Queue and gets the details from the downstream Exchange.
// Pushes the details to S3 and marks as done from the Queue.
func ProcessTransactions(awsConfig *aws.Config, context *context.Context) error {
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
	sqsQueue := os.Getenv(dcaConfig.EnvSQSPendingOrdersQueue)

	sqsAttributes := []*string{&exchangeAttribute, &realAttribute}
	sqsRecieveMessage := &sqs.ReceiveMessageInput{
		QueueUrl:              &sqsQueue,
		MessageAttributeNames: aws.ToStringSlice(sqsAttributes),
		VisibilityTimeout:     60,
	}

	// Get Messages from Queue
	log.Infof("Getting Messages from %s", sqsQueue)
	sqsResponse, err := sqsClient.ReceiveMessage(*context, sqsRecieveMessage)
	if err != nil {
		return err
	}

	if len(sqsResponse.Messages) == 0 {
		log.Warn("No SQS Messages found, returning")
		return nil
	}

	// For each of the SQS Messages
	for _, message := range sqsResponse.Messages {
		log.Infof("Processing SQS Message: %s", *message.MessageId)

		// Extract Details from the Message
		exchange := message.MessageAttributes[exchangeAttribute]
		realAtt := message.MessageAttributes[realAttribute]

		// If the message is a fake/testing message, mark as deleted and continue
		if *realAtt.StringValue == "false" {
			log.Warnf("Recieved SQS message which was not real. Deleting MessageId %s", *message.MessageId)
			sqsClient.DeleteMessage(*context, &sqs.DeleteMessageInput{
				QueueUrl:      &sqsQueue,
				ReceiptHandle: message.ReceiptHandle,
			})

			continue
		}

		if exchange.StringValue == nil {
			log.Warnf("Recieved SQS message with no Exchange Set. Skipping MessageId: %s", *message.MessageId)
			continue
		}

		messageBytes := []byte(*message.Body)

		var po orders.PendingOrders
		err := json.Unmarshal(messageBytes, &po)
		if err != nil {
			log.Error("Unable to unmarshal json from Message %s", *message.MessageId)
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
		s3Path := os.Getenv(dcaConfig.EnvS3ProcessedTransaction)

		for _, order := range *orders {

			if order.TransactionId == "" {
				log.Warnf("Found an order with no transaction id: %s", order)
				continue
			}

			s3Path := fmt.Sprintf(
				"%s/exchange=%s/%s.json",
				s3Path,
				strings.ToLower(*exchange.StringValue),
				order.TransactionId,
			)

			orderBytes, err := json.Marshal(order)
			if err != nil {
				return err
			}

			log.Infof("Uploading Transaction %s to Bucket %s, Key %s", order.TransactionId, s3Bucket, s3Path)
			s3Client.PutObject(*context, &s3.PutObjectInput{
				Bucket: &s3Bucket,
				Key:    &s3Path,
				Body:   bytes.NewReader(orderBytes),
			})
		}

		// Delete from Queue
		log.Infof("Deleting MessageId %s", *message.MessageId)
		sqsClient.DeleteMessage(*context, &sqs.DeleteMessageInput{
			QueueUrl:      &sqsQueue,
			ReceiptHandle: message.ReceiptHandle,
		})
	}

	return nil
}
