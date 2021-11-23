package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	krakenapi "github.com/beldur/kraken-go-api-client"
	"github.com/kiran94/dca-manager/configuration"
	"github.com/kiran94/dca-manager/orders"
	log "github.com/sirupsen/logrus"
)

const (
	envLambdaServerPort = "_LAMBDA_SERVER_PORT"
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetReportCaller(false)
	log.Info("Lambda Execution Starting")

	lambdaPort := os.Getenv(envLambdaServerPort)

	if lambdaPort != "" {
		log.SetFormatter(&log.TextFormatter{
			DisableColors: true,
			FullTimestamp: true,
		})
		log.SetFormatter(&log.JSONFormatter{})

		lambda.Start(HandleRequest)

	} else {
		log.SetFormatter(&log.TextFormatter{
			DisableColors: false,
			FullTimestamp: true,
		})

		res, err := HandleRequest(context.TODO(), MyEvent{Name: "Event"})

		fmt.Println(res)
		fmt.Println(err)
	}

	log.Info("Lambda Execution Done.")
}

type MyEvent struct {
	Name string `json:"name"`
}

func HandleRequest(c context.Context, event MyEvent) (string, error) {
	log.Info("Starting Handling Request")

	// Call AWS
	config, err := config.LoadDefaultConfig(c)
	if err != nil {
		return "", err
	}

	// Get DCA Configuration
	dcaConfig, err := configuration.GetDCAConfiguration(config, c)
	if err != nil {
		return "", err
	}
	log.Debug(dcaConfig)

	// Call Kraken API
	key, secret, ssmErr := configuration.GetKrakenDetails(config, c)
	if ssmErr != nil {
		return "", ssmErr
	}

	// Place orders
	o := map[string]orders.Orderer{}
	o["kraken"] = orders.KrakenOrderer{
		Client: krakenapi.New(*key, *secret),
	}

	for index, order := range dcaConfig.Orders {
		log.Debugf("Running Order %s with Exchange %s", index, order.Exchange)

		// REAL
		exchange := o[order.Exchange]
		orderResult, orderErr := exchange.MakeOrder(&order)

		// FAKE

		// orderResult := &orders.OrderFufilled{
		// 	Result: &krakenapi.AddOrderResponse{
		// 		TransactionIds: []string{"TXID"},
		// 		Description: krakenapi.OrderDescription{
		// 			AssetPair:      "ADAGBP",
		// 			Close:          "100",
		// 			Leverage:       "Leverage",
		// 			Order:          "Order",
		// 			OrderType:      "OrderType",
		// 			PrimaryPrice:   "PrimaryPrice",
		// 			SecondaryPrice: "SecondaryPrice",
		// 			Type:           "Type",
		// 		},
		// 	},
		// 	Timestamp: 12345678,
		// }

		// var orderErr error

		////////////////////////////

		if orderErr != nil {
			return "", orderErr
		}

		const s3transactionPrefix string = "transactions"
		orderResultInner := (*orderResult).Result.(*krakenapi.AddOrderResponse)

		s3Path := fmt.Sprintf(
			"%s/ordertype=%s/pair=%s/%d.json",
			s3transactionPrefix,
			orderResultInner.Description.OrderType,
			orderResultInner.Description.AssetPair,
			(*orderResult).Timestamp)

		orderResultBytes, err := json.Marshal(orderResult)
		if err != nil {
			return "", err
		}

		s3Bucket := os.Getenv(configuration.EnvS3Bucket)
		s3Client := s3.NewFromConfig(config)

		log.Infof("Uploading to Bucket %s, Key %s", s3Bucket, s3Path)
		s3Client.PutObject(c, &s3.PutObjectInput{
			Bucket: &s3Bucket,
			Key:    &s3Path,
			Body:   bytes.NewReader(orderResultBytes),
		})
	}

	return fmt.Sprintf("Hello %s!", event.Name), nil
}
