package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	awsEvents "github.com/aws/aws-lambda-go/events"
	awsLambda "github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
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

// DCAServices contains all services to be injected into logic.
type DCAServices struct {
	awsConfig             aws.Config
	s3Access              pkg.S3Access
	ssmAccess             pkg.SSMAccess
	sqsAccess             pkg.SQSAccess
	configSource          configuration.DCAConfigurationSource
	ordererFactory        orders.OrdererFactory
	pendingOrderSubmitter orders.PendingOrderQueue
}

// AppConfig contains all configuration to be injected into logic
type AppConfig struct {
	s3bucket      string
	dcaConfigPath string
	allowReal     bool
	transactions  struct {
		pendingS3TransactionPrefix   string
		processedS3TransactionPrefix string
	}
	queue struct {
		sqsURL string
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
	appConfig.queue.sqsURL = os.Getenv(configuration.EnvSQSPendingOrdersQueue)
	appConfig.glue.processTransactionJob = os.Getenv(configuration.EnvGlueProcessTransactionJob)
	appConfig.glue.processTransactionOperation = os.Getenv(configuration.EnvGlueProcessTransactionOperation)
}

func main() {
	logrus.SetOutput(os.Stdout)
	logrus.SetReportCaller(false)
	logrus.Info("Lambda Execution Starting")

	if os.Getenv("_LAMBDA_SERVER_PORT") != "" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
		awsLambda.Start(handleRequest)

	} else {
		logrus.SetFormatter(&logrus.TextFormatter{})
		handleRequestLocally()
	}

	logrus.Info("Lambda Execution Done.")
}

func handleRequest(c context.Context, event awsEvents.CloudWatchEvent) (*string, error) {
	pendingOrders, err := ExecuteOrders(c, dcaServices, appConfig)
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

// ExecuteOrders will execute orders from the DCA configuration
// into exchanges
func ExecuteOrders(ctx context.Context, services *DCAServices, config *AppConfig) (*[]orders.PendingOrders, error) {
	logrus.Info("Executing Orders")

	// Get DCA Configuration
	logrus.WithFields(logrus.Fields{
		"s3bucket": config.s3bucket,
		"s3path":   config.dcaConfigPath,
	}).Info("Getting DCA Configuration")

	dcaConf, err := services.configSource.GetDCAConfiguration(ctx, services.s3Access, &config.s3bucket, &config.dcaConfigPath)
	if err != nil {
		return nil, err
	}
	logrus.WithField("config", *dcaConf).Debug("Pulled config")

	logrus.Info("Getting Orderers")
	o, ordererErr := services.ordererFactory.GetOrderers(ctx, services.ssmAccess)
	if ordererErr != nil {
		return nil, ordererErr
	}

	// Execute Orders
	submittedPendingOrders := make([]orders.PendingOrders, len(dcaConf.Orders))
	for index, order := range dcaConf.Orders {
		logrus.WithFields(logrus.Fields{
			"index":     index,
			"exchange":  order.Exchange,
			"pair":      order.Pair,
			"volume":    order.Volume,
			"type":      order.OrderType,
			"direction": order.Direction,
		}).Info("Executing Order")

		var orderResult *orders.OrderFufilled
		var orderErr error

		if config.allowReal {
			exchange, ok := (*o)[order.Exchange]
			if !ok {
				return nil, fmt.Errorf("no orderer found for exchange %s", order.Exchange)
			}

			orderResult, orderErr = exchange.MakeOrder(&order)
		} else {
			orderResult, orderErr = orders.GetFakeOrderFufilled()
		}

		if orderErr != nil {
			return nil, orderErr
		}

		s3Path := fmt.Sprintf(
			"%s/exchange=%s/%s.json",
			config.transactions.pendingS3TransactionPrefix,
			strings.ToLower(order.Exchange),
			orderResult.TransactionID,
		)

		orderResultBytes, err := json.Marshal(orderResult)
		if err != nil {
			return nil, err
		}

		logrus.WithFields(logrus.Fields{
			"s3bucket":      config.s3bucket,
			"s3path":        s3Path,
			"transactionId": orderResult.TransactionID,
		}).Info("Uploading Order result to bucket")

		_, err = services.s3Access.PutObject(ctx, &s3.PutObjectInput{
			Bucket: &config.s3bucket,
			Key:    &s3Path,
			Body:   bytes.NewReader(orderResultBytes),
		})
		if err != nil {
			return nil, err
		}

		// Submit to SQS
		po := orders.PendingOrders{
			TransactionID: orderResult.TransactionID,
			S3Bucket:      config.s3bucket,
			S3Key:         s3Path,
		}

		submitErr := services.pendingOrderSubmitter.SubmitPendingOrder(ctx, services.sqsAccess, &po, order.Exchange, config.allowReal, config.queue.sqsURL)
		if submitErr != nil {
			return nil, submitErr
		}

		submittedPendingOrders[index] = po
	}

	return &submittedPendingOrders, nil
}

func handleRequestLocally() {
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

	res, err := handleRequest(context.Background(), event)
	if res != nil {
		logrus.WithField("result", *res).Info("request successful locally.")
	}

	if err != nil {
		logrus.WithError(err).Error("Error running request locally.")
	}
}
