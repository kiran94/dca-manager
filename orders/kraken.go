package orders

import (
	"time"

	"github.com/beldur/kraken-go-api-client"
	config "github.com/kiran94/dca-manager/configuration"
	log "github.com/sirupsen/logrus"
)

type KrakenOrderer struct {
	Client *krakenapi.KrakenAPI
}

func (ko *KrakenOrderer) New(client *krakenapi.KrakenApi) {
	ko.Client = client
}

func (ko KrakenOrderer) MakeOrder(order *config.DCAOrder) (*OrderFufilled, error) {

	log.Infof("Making Order: %s %s %s (%s)", order.Direction, order.Volume, order.Pair, order.OrderType)

	if !order.Enabled {
		log.Warn("order disabled, skipping")
		return nil, nil
	}

	addOrderResponse, err := ko.Client.AddOrder(order.Pair, order.Direction, order.OrderType, order.Volume, make(map[string]string, 0))
	if err != nil {
		return nil, err
	}

	log.Infof("Order Response: %s", addOrderResponse)

	o := OrderFufilled{}
	o.Result = addOrderResponse
	o.Timestamp = time.Now().Unix()
	return &o, nil
}
