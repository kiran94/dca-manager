package main

import (
	"context"
	"errors"
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

		event := lambda.LambdaEvent{
			Operation: "ExecuteOrders",
		}

		res, err := HandleRequest(context.TODO(), event)
		fmt.Printf("Result: %s \n", *res)
		fmt.Printf("Error: %s \n", err)
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
	} else {
		err = errors.New(fmt.Sprintf("Unconfigured operation: %s", event.Operation))
	}

	if err != nil {
		return nil, err
	}

	res := fmt.Sprintf("Operation %s completed successfully.", event.Operation)
	return &res, nil
}
