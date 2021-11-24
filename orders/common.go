package orders

import (
	config "github.com/kiran94/dca-manager/configuration"
)

// Responsible for making DCA orders to an Exchange.
type Orderer interface {
	MakeOrder(order *config.DCAOrder) (*OrderFufilled, error)
}

type OrderFufilled struct {
	TransactionId string      `json:"transaction_id"`
	Timestamp     int64       `json:"timestamp"`
	Result        interface{} `json:"result"`
}
