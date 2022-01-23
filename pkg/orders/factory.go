package orders

import (
	"context"

	"github.com/kiran94/dca-manager/pkg"
	"github.com/kiran94/dca-manager/pkg/configuration"

	krakenapi "github.com/beldur/kraken-go-api-client"
)

// OrdererFactory is an abstraction to get Exchange Orderers
type OrdererFactory interface {
	GetOrderers(ctx context.Context, ssm pkg.SSMAccess) (*map[string]Orderer, error)
}

// OrdererFac is responsible for getting Exchange Orderers
type OrdererFac struct{}

// GetOrderers gets a map of exchange to Orderer.
func (o OrdererFac) GetOrderers(ctx context.Context, ssm pkg.SSMAccess) (*map[string]Orderer, error) {
	orderers := map[string]Orderer{}

	// Kraken
	krakenConfig := configuration.KrakenConf{}
	key, secret, err := krakenConfig.GetKrakenDetails(ctx, ssm)
	if err != nil {
		return nil, err
	}

	orderers["kraken"] = KrakenOrderer{
		Client: krakenapi.New(*key, *secret),
	}

	return &orderers, nil
}
