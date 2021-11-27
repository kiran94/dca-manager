package configuration

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

var (
	SSMKrakenKey    string = "/dca-manager/kraken/key"
	SSMKrakenSecret string = "/dca-manager/kraken/secret"
)

// Gets the Kraken Key and Secret from AWS SSM
func GetKrakenDetails(config aws.Config, context context.Context) (key *string, secret *string, error error) {
	ssmClient := ssm.NewFromConfig(config)

	krakenKey, err := ssmClient.GetParameter(context, &ssm.GetParameterInput{
		Name:           &SSMKrakenKey,
		WithDecryption: true,
	})

	if err != nil {
		return nil, nil, err
	}

	krakenSecret, err2 := ssmClient.GetParameter(context, &ssm.GetParameterInput{
		Name:           &SSMKrakenSecret,
		WithDecryption: true,
	})

	if err2 != nil {
		return nil, nil, err2
	}

	return krakenKey.Parameter.Value, krakenSecret.Parameter.Value, nil
}

