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
	krakenapi "github.com/beldur/kraken-go-api-client"
	dcsConfig "github.com/kiran94/dca-manager/configuration"
	"github.com/kiran94/dca-manager/orders"
	log "github.com/sirupsen/logrus"
)

const (
	s3transactionPrefix string = "transactions/pending"
)

// Executes the configured orders.
func ExecuteOrders(awsConfig *aws.Config, context *context.Context) error {

	log.Info("Executing Orders")

	// Get DCA Configuration
	dcaConfig, err := dcsConfig.GetDCAConfiguration(*awsConfig, *context)
	if err != nil {
		return err
	}
	log.Debug(dcaConfig)

	// Call Kraken API
	key, secret, ssmErr := dcsConfig.GetKrakenDetails(*awsConfig, *context)
	if ssmErr != nil {
		return ssmErr
	}

	// Create Orders (per Exchange)
	o := map[string]orders.Orderer{}
	o["kraken"] = orders.KrakenOrderer{
		Client: krakenapi.New(*key, *secret),
	}

	// Execute Orders
	for index, order := range dcaConfig.Orders {
		log.Debugf("Running Order %s with Exchange %s", index, order.Exchange)

		var orderResult *orders.OrderFufilled
		var orderErr error

		if os.Getenv(dcsConfig.EnvAllowReal) != "" {
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

		s3Bucket := os.Getenv(dcsConfig.EnvS3Bucket)
		s3Client := s3.NewFromConfig(*awsConfig)

		log.Infof("Uploading to Bucket %s, Key %s", s3Bucket, s3Path)
		s3Client.PutObject(*context, &s3.PutObjectInput{
			Bucket: &s3Bucket,
			Key:    &s3Path,
			Body:   bytes.NewReader(orderResultBytes),
		})
	}

	return nil
}
