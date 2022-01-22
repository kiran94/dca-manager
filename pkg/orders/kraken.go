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

type KrakenAccess interface {
	AddOrder(pair string, direction string, orderType string, volume string, args map[string]string) (*krakenapi.AddOrderResponse, error)
	QueryOrders(txids string, args map[string]string) (*krakenapi.QueryOrdersResponse, error)
}

type KrakenOrderer struct {
	Client KrakenAccess
}

func (ko *KrakenOrderer) New(client KrakenAccess) {
	ko.Client = client
}

// Execute the given DCA Order on the Kraken Exchange
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
	o.TransactionId = addOrderResponse.TransactionIds[0]
	return &o, nil
}

// Processes the given Transactions.
// For a given transaction, reach out to Kraken and get the order
// information and standardise into the standard
// OrderComplete object
func (ko KrakenOrderer) ProcessTransaction(transactionId ...string) (*[]OrderComplete, error) {
	if len(transactionId) == 0 {
		return nil, errors.New("no transactions provided")
	}

	txids := strings.Join(transactionId, ",")
	args := make(map[string]string, 1)

	logrus.WithField("transactionId", txids).Info("Getting Details for Transactions")
	transactions, err := ko.Client.QueryOrders(txids, args)
	if err != nil {
		return nil, err
	}

	completeOrders := make([]OrderComplete, len(*transactions))

	logrus.Info("Mapping back response to transactions")
	index := 0
	for transactionId := range *transactions {
		logrus.WithField("transactionId", transactionId).Debug("Mapping Transaction")

		co := (*transactions)[transactionId]
		orderComplete := OrderComplete{
			TransactionId:  transactionId,
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
			"transactionId": transactionId,
			"orderComplete": orderComplete,
		}).Debug("Complete Order")

		completeOrders[index] = orderComplete
		index += 1
	}

	return &completeOrders, nil
}
