package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	awsLambda "github.com/aws/aws-lambda-go/lambda"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/kiran94/dca-manager/lambda"
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
		// Running in Lambda
		log.SetFormatter(&log.TextFormatter{
			DisableColors: true,
			FullTimestamp: true,
		})
		log.SetFormatter(&log.JSONFormatter{})

		awsLambda.Start(HandleRequest)

	} else {
		// Running Locally
		log.SetFormatter(&log.TextFormatter{
			DisableColors: false,
			FullTimestamp: true,
		})

		operation := flag.String("operation", "ExecuteOrders", "The Operation to Execute")
		flag.Parse()

		event := lambda.LambdaEvent{
			// Operation: "ExecuteOrders",
			// Operation: "ProcessTransaction",
			Operation: *operation,
		}

		res, err := HandleRequest(context.TODO(), event)
		if res != nil {
			fmt.Printf("Result: %s \n", *res)
		} else {
			fmt.Println("Result was nil")
		}

		if err != nil {
			fmt.Printf("Error: %s \n", err)
		} else {
			log.Debug("Error was null")
		}
	}

	log.Info("Lambda Execution Done.")
}

func HandleRequest(c context.Context, event lambda.LambdaEvent) (*string, error) {
	log.Info("Starting Handling Request")

	awsConfig, err := awsConfig.LoadDefaultConfig(c)
	if err != nil {
		return nil, err
	}

	if event.Operation == "ExecuteOrders" {
		err = lambda.ExecuteOrders(&awsConfig, &c)
	} else if event.Operation == "ProcessTransaction" {
		err = lambda.ProcessTransactions(&awsConfig, &c)
	} else {
		err = fmt.Errorf("Unconfigured operation: %s", event.Operation)
	}

	if err != nil {
		return nil, err
	}

	res := fmt.Sprintf("Operation %s completed successfully.", event.Operation)
	return &res, nil
}
