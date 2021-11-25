package lambda

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	krakenapi "github.com/beldur/kraken-go-api-client"
	dcaConfig "github.com/kiran94/dca-manager/configuration"
	"github.com/kiran94/dca-manager/orders"
	log "github.com/sirupsen/logrus"
)

const (
	s3transactionPrefix string = "transactions/status=pending"
)

// Executes the configured orders.
func ExecuteOrders(awsConfig *aws.Config, context *context.Context) error {

	log.Info("Executing Orders")

	// Get DCA Configuration
	dcaConf, err := dcaConfig.GetDCAConfiguration(*awsConfig, *context)
	if err != nil {
		return err
	}
	log.Debug(dcaConf)

	// Call Kraken API
	key, secret, ssmErr := dcaConfig.GetKrakenDetails(*awsConfig, *context)
	if ssmErr != nil {
		return ssmErr
	}

	// Create Orders (per Exchange)
	o := map[string]orders.Orderer{}
	o["kraken"] = orders.KrakenOrderer{
		Client: krakenapi.New(*key, *secret),
	}

	// Execute Orders
	s3Client := s3.NewFromConfig(*awsConfig)
	sqsClient := sqs.NewFromConfig(*awsConfig)

	for index, order := range dcaConf.Orders {
		log.Debugf("Running Order %s with Exchange %s", index, order.Exchange)

		var orderResult *orders.OrderFufilled
		var orderErr error
		allowReal := os.Getenv(dcaConfig.EnvAllowReal) != ""

		if allowReal {
			exchange := o[order.Exchange]
			orderResult, orderErr = exchange.MakeOrder(&order)
		} else {
			orderResult, orderErr = orders.GetFakeOrderFufilled()
		}

		if orderErr != nil {
			return orderErr
		}

		s3Path := fmt.Sprintf(
			"%s/exchange=%s/%s.json",
			s3transactionPrefix,
			strings.ToLower(order.Exchange),
			orderResult.TransactionId,
		)

		orderResultBytes, err := json.Marshal(orderResult)
		if err != nil {
			return err
		}

		s3Bucket := os.Getenv(dcaConfig.EnvS3Bucket)

		log.Infof("Uploading to Bucket %s, Key %s", s3Bucket, s3Path)
		s3Client.PutObject(*context, &s3.PutObjectInput{
			Bucket: &s3Bucket,
			Key:    &s3Path,
			Body:   bytes.NewReader(orderResultBytes),
		})

		// Submit to SQS
		po := orders.PendingOrders{
			TransactionId: orderResult.TransactionId,
			S3Bucket:      s3Bucket,
			S3Key:         s3Path,
		}

		sqsMessageBodyBytes, err := json.Marshal(po)
		if err != nil {
			return err
		}

		sqsMessage := string(sqsMessageBodyBytes)
		sqsQueue := os.Getenv(dcaConfig.EnvSQSPendingOrdersQueue)
		sqsMessageInput := &sqs.SendMessageInput{
			QueueUrl:    &sqsQueue,
			MessageBody: aws.String(sqsMessage),
			MessageAttributes: map[string]types.MessageAttributeValue{
				"Exchange": {
					DataType:    aws.String("String"),
					StringValue: aws.String(order.Exchange),
				},
				"TransactionId": {
					DataType:    aws.String("String"),
					StringValue: aws.String(orderResult.TransactionId),
				},
				"Real": {
					DataType:    aws.String("String"),
					StringValue: aws.String(strconv.FormatBool(allowReal)),
				},
			},
		}

		log.Infof("Submitting Transaction %s to Queue %s", po.TransactionId, sqsQueue)
		sqsClient.SendMessage(*context, sqsMessageInput)
	}

	return nil
}
