package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/beldur/kraken-go-api-client"
	"github.com/kiran94/dca-manager/configuration"
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

		HandleRequest(context.TODO(), MyEvent{Name: "Kiran"})
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
		log.Fatal(err)
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

	// PLACE ORDERS
	kraken := krakenapi.New(*key, *secret)
	for index, order := range dcaConfig.Orders {

		log.Infof("Order %d: %s %s %s (%s)", index, order.Direction, order.Volume, order.Pair, order.OrderType)

		if !order.Enabled {
			log.Warn("order disabled, skipping")
			continue
		}

		addOrderResponse, err := kraken.AddOrder(order.Pair, order.Direction, order.OrderType, order.Volume, make(map[string]string, 0))
		if err != nil {
			panic(err)
		}

		fmt.Println(addOrderResponse)
	}

	return fmt.Sprintf("Hello %s!", event.Name), nil
}
