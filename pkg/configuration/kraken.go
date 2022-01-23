package configuration

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/kiran94/dca-manager/pkg"
)

// SSM Keys to fetch for Kraken to connect to Kraken.
var (
	SSMKrakenKey    string = "/dca-manager/kraken/key"
	SSMKrakenSecret string = "/dca-manager/kraken/secret"
)

// KrakenConfiguration is an abstraction to gets details to connect to the Kraken Exchange.
type KrakenConfiguration interface {
	GetKrakenDetails(ctx context.Context, ssmClient pkg.SSMAccess) (key *string, secret *string, error error)
}

// KrakenConf gets details to connect to the Kraken Exchange.
type KrakenConf struct{}

// GetKrakenDetails gets the Kraken Key and Secret.
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
