package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	awsEvents "github.com/aws/aws-lambda-go/events"
	awsLambda "github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	krakenapi "github.com/beldur/kraken-go-api-client"
	dcaConfig "github.com/kiran94/dca-manager/configuration"
	"github.com/kiran94/dca-manager/orders"
	log "github.com/sirupsen/logrus"
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


func HandleRequest(c context.Context, event awsEvents.CloudWatchEvent) (*string, error) {
	awsConfig, err := awsConfig.LoadDefaultConfig(c)
	if err != nil {
		return nil, err
	}

	pendingOrders, err := ExecuteOrders(&awsConfig, &c)
	if err != nil {
		return nil, err
	}

	serialisedOrders, err := json.Marshal(*pendingOrders)
	if err != nil {
		return nil, err
	}

	serialisedOrdersString := string(serialisedOrders)
	return &serialisedOrdersString, err
}

// Executes the configured orders.
func ExecuteOrders(awsConfig *aws.Config, context *context.Context) (*[]orders.PendingOrders, error) {
	log.Info("Executing Orders")

	// Get DCA Configuration
	dcaConf, err := dcaConfig.GetDCAConfiguration(*awsConfig, *context)
	if err != nil {
		return nil, err
	}
	log.Debug(dcaConf)

	// Call Kraken API
	key, secret, ssmErr := dcaConfig.GetKrakenDetails(*awsConfig, *context)
	if ssmErr != nil {
		return nil, ssmErr
	}

	// Create Orders (per Exchange)
	o := map[string]orders.Orderer{}
	o["kraken"] = orders.KrakenOrderer{
		Client: krakenapi.New(*key, *secret),
	}

	// Execute Orders
	s3Client := s3.NewFromConfig(*awsConfig)
	sqsClient := sqs.NewFromConfig(*awsConfig)

	submittedPendingOrders := make([]orders.PendingOrders, len(dcaConf.Orders))

	for index, order := range dcaConf.Orders {
		log.Debugf("Running Order %s with Exchange %s", index, order.Exchange)

		var orderResult *orders.OrderFufilled
		var orderErr error
		allowReal := os.Getenv(dcaConfig.EnvAllowReal) != ""
		s3transactionPrefix := os.Getenv(dcaConfig.EnvS3PendingTransaction)

		if allowReal {
			exchange := o[order.Exchange]
			orderResult, orderErr = exchange.MakeOrder(&order)
		} else {
			orderResult, orderErr = orders.GetFakeOrderFufilled()
		}

		if orderErr != nil {
			return nil, orderErr
		}

		s3Path := fmt.Sprintf(
			"%s/exchange=%s/%s.json",
			s3transactionPrefix,
			strings.ToLower(order.Exchange),
			orderResult.TransactionId,
		)

		orderResultBytes, err := json.Marshal(orderResult)
		if err != nil {
			return nil, err
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

		submitErr := SubmitPendingOrder(sqsClient, &po, context, order.Exchange, allowReal)
		if submitErr != nil {
			return nil, submitErr
		}

		submittedPendingOrders = append(submittedPendingOrders, po)
	}

	return &submittedPendingOrders, nil
}

// Submits the given pending order to queue.
func SubmitPendingOrder(sc *sqs.Client, po *orders.PendingOrders, c *context.Context, exchange string, real bool) error {
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
				StringValue: aws.String(exchange),
			},
			"TransactionId": {
				DataType:    aws.String("String"),
				StringValue: aws.String(po.TransactionId),
			},
			"Real": {
				DataType:    aws.String("String"),
				StringValue: aws.String(strconv.FormatBool(real)),
			},
		},
	}

	log.Infof("Submitting Transaction %s to Queue %s", po.TransactionId, sqsQueue)
	_, sqsErr := sc.SendMessage(*c, sqsMessageInput)

	if sqsErr != nil {
		return sqsErr
	}

	return nil
}

func HandleRequestLocally() {
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

	res, err := HandleRequest(context.TODO(), event)

	if res != nil {
		log.Infof("Result: %s \n", *res)
	}

	if err != nil {
		log.Errorf("Error: %s \n", err)
	}
}
