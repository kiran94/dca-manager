package orders

import (
	"errors"
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

	if len(addOrderResponse.TransactionIds) > 1 {
		log.Warnf("Recieved more then one TransactionIds %s", addOrderResponse.TransactionIds)
	} else if len(addOrderResponse.TransactionIds) == 0 {
		return nil, errors.New("No Transactions ids recieved")
	}

	o := OrderFufilled{}
	o.Result = addOrderResponse
	o.Timestamp = time.Now().Unix()
	o.TransactionId = addOrderResponse.TransactionIds[0]
	return &o, nil
}
