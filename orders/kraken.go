package orders

import (
	"errors"
	"strings"
	"time"

	"github.com/beldur/kraken-go-api-client"
	config "github.com/kiran94/dca-manager/configuration"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

type KrakenOrderer struct {
	Client *krakenapi.KrakenAPI
}

func (ko *KrakenOrderer) New(client *krakenapi.KrakenApi) {
	ko.Client = client
}

// Execute the given DCA Order on the Kraken Exchange
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

// Processes the given Transactions.
// For a given transaction, reach out to Kraken and get the order
// information and standardise inot the standard
// OrderComplete object
func (ko KrakenOrderer) ProcessTransaction(transactionId ...string) (*[]OrderComplete, error) {

	txids := strings.Join(transactionId, ",")
	args := make(map[string]string, 1)

	log.Infof("Getting Details for Transactions %s", txids)
	transactions, err := ko.Client.QueryOrders(txids, args)
	if err != nil {
		return nil, err
	}

	completeOrders := make([]OrderComplete, 1)

	log.Info("Mapping back response to transactions")
	for transactionId := range *transactions {
		log.Debugf("Mapping %s", transactionId)

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

		log.Debugf("Complete Order: Transaction %s: Order: %s \n", transactionId, orderComplete)
		completeOrders = append(completeOrders, orderComplete)
	}

	return &completeOrders, nil
}
