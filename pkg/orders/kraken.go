package orders

import (
	"errors"
	"strings"
	"time"

	krakenapi "github.com/beldur/kraken-go-api-client"
	config "github.com/kiran94/dca-manager/pkg/configuration"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

// KrakenAccess is an abstraction that provides access to the Kraken Exchange.
type KrakenAccess interface {
	AddOrder(pair string, direction string, orderType string, volume string, args map[string]string) (*krakenapi.AddOrderResponse, error)
	QueryOrders(txids string, args map[string]string) (*krakenapi.QueryOrdersResponse, error)
}

// KrakenOrderer providess access to the Kraken Exchange
type KrakenOrderer struct {
	Client KrakenAccess
}

// New creates an entirely new access object to Kraken.
func (ko *KrakenOrderer) New(client KrakenAccess) {
	ko.Client = client
}

// MakeOrder executes the provided DCAOrder on the Kraken Exchange.
func (ko KrakenOrderer) MakeOrder(order *config.DCAOrder) (*OrderFufilled, error) {

	logrus.WithFields(logrus.Fields{
		"direction": order.Direction,
		"volume":    order.Volume,
		"pair":      order.Pair,
		"type":      order.OrderType,
		"exchange":  order.Exchange,
		"enabled":   order.Enabled,
	}).Info("Making Order")

	if !order.Enabled {
		logrus.Warn("order disabled, skipping")
		return nil, nil
	}

	addOrderResponse, err := ko.Client.AddOrder(order.Pair, order.Direction, order.OrderType, order.Volume, make(map[string]string, 0))
	if err != nil {
		return nil, err
	}

	logrus.WithField("transactionId", addOrderResponse.TransactionIds).Info("Order Response")

	if len(addOrderResponse.TransactionIds) > 1 {
		logrus.Warnf("Received more then one TransactionIds %s", addOrderResponse.TransactionIds)
	} else if len(addOrderResponse.TransactionIds) == 0 {
		return nil, errors.New("no transactions ids received")
	}

	o := OrderFufilled{}
	o.Result = addOrderResponse
	o.Timestamp = time.Now().Unix()
	o.TransactionID = addOrderResponse.TransactionIds[0]
	return &o, nil
}

// ProcessTransaction takes the given transactionIds
// and loads details for them from the Kraken Exchange
// and standardise the order into a OrderComplete object
func (ko KrakenOrderer) ProcessTransaction(transactionID ...string) (*[]OrderComplete, error) {
	if len(transactionID) == 0 {
		return nil, errors.New("no transactions provided")
	}

	txids := strings.Join(transactionID, ",")
	args := make(map[string]string, 1)

	logrus.WithField("transactionId", txids).Info("Getting Details for Transactions")
	transactions, err := ko.Client.QueryOrders(txids, args)
	if err != nil {
		return nil, err
	}

	completeOrders := make([]OrderComplete, len(*transactions))

	logrus.Info("Mapping back response to transactions")
	index := 0
	for transactionID := range *transactions {
		logrus.WithField("transactionId", transactionID).Debug("Mapping Transaction")

		co := (*transactions)[transactionID]
		orderComplete := OrderComplete{
			TransactionID:  transactionID,
			ExchangeStatus: co.Status,
			Pair:           co.Description.AssetPair,
			OrderType:      co.Description.OrderType,
			Type:           co.Description.Type,
			Price:          decimal.NewFromFloat(co.Price),
			Fee:            decimal.NewFromFloat(co.Fee),
			Volume:         decimal.NewFromFloat(co.VolumeExecuted),
			OpenTime:       co.OpenTime,
			CloseTime:      co.CloseTime,
		}

		logrus.WithFields(logrus.Fields{
			"transactionId": transactionID,
			"orderComplete": orderComplete,
		}).Debug("Complete Order")

		completeOrders[index] = orderComplete
		index++
	}

	return &completeOrders, nil
}
