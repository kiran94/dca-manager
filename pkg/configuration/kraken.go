package configuration

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/kiran94/dca-manager/pkg"
)

var (
	SSMKrakenKey    string = "/dca-manager/kraken/key"
	SSMKrakenSecret string = "/dca-manager/kraken/secret"
)

type KrakenConfiguration interface {
	GetKrakenDetails(ctx context.Context, ssmClient pkg.SSMAccess) (key *string, secret *string, error error)
}

type KrakenConf struct{}

// Gets the Kraken Key and Secret from AWS SSM
func (k KrakenConf) GetKrakenDetails(ctx context.Context, ssmClient pkg.SSMAccess) (key *string, secret *string, error error) {
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
