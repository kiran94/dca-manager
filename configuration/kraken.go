package configuration

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

var (
	SSMKrakenKey    string = "/dca-manager/kraken/key"
	SSMKrakenSecret string = "/dca-manager/kraken/secret"
)

type SSMAccess interface {
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

type SSM struct {
	Client *ssm.Client
}

func (s *SSM) GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	return s.Client.GetParameter(ctx, params, optFns...)
}

// Gets the Kraken Key and Secret from AWS SSM
func GetKrakenDetails(ctx context.Context, ssmClient SSMAccess) (key *string, secret *string, error error) {
	krakenKey, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           &SSMKrakenKey,
		WithDecryption: true,
	})

	if err != nil {
		return nil, nil, err
	}

	krakenSecret, err2 := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           &SSMKrakenSecret,
		WithDecryption: true,
	})

	if err2 != nil {
		return nil, nil, err2
	}

	return krakenKey.Parameter.Value, krakenSecret.Parameter.Value, nil
}
