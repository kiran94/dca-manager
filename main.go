package main

import (
	"context"
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

		res, err := HandleRequest(context.TODO(), MyEvent{Name: "Event"})
		fmt.Println(res)
		fmt.Println(err)
	}

	log.Info("Lambda Execution Done.")
}

type MyEvent struct {
	Name string `json:"name"`
}

func HandleRequest(c context.Context, event MyEvent) (*string, error) {
	log.Info("Starting Handling Request")

	awsConfig, err := awsConfig.LoadDefaultConfig(c)
	if err != nil {
		return nil, err
	}

	err = lambda.ExecuteOrders(&awsConfig, &c)
	if err != nil {
		return nil, err
	}

    res := fmt.Sprintf("Hello %s!", event.Name)
	return &res, nil
}
