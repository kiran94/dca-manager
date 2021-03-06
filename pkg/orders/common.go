package orders

import (
	config "github.com/kiran94/dca-manager/pkg/configuration"
	"github.com/shopspring/decimal"
)

// Orderer Responsible for making DCA orders to an Exchange.
type Orderer interface {
	MakeOrder(order *config.DCAOrder) (*OrderFufilled, error)
	ProcessTransaction(transactionsIds ...string) (*[]OrderComplete, error)
}

// OrderFufilled which has been sent to the Exchange
type OrderFufilled struct {
	TransactionID string      `json:"transaction_id"`
	Timestamp     int64       `json:"timestamp"`
	Result        interface{} `json:"result"`
}

// PendingOrders which is processing on the exchange
// where the s3 bucket & key define the result from the
// exchange from the initial call.
//
// An order could be accepted by the exchange but not necessarily
// successful yet so the payload in s3 is whatever they sent back to us.
//
// This object is used to push the transaction to an out-of-process
// queue for later processing
type PendingOrders struct {
	TransactionID string `json:"transaction_id"`
	S3Bucket      string `json:"s3_bucket"`
	S3Key         string `json:"s3_key"`
}

// OrderComplete from an Exchange
// This object acts as a common abstraction
// amongst all exchanges
type OrderComplete struct {
	TransactionID  string          `json:"transaction_id"`
	ExchangeStatus string          `json:"exchange_status"`
	Pair           string          `json:"pair"`
	OrderType      string          `json:"order_type"`
	Type           string          `json:"type"`
	Price          decimal.Decimal `json:"price"`
	Fee            decimal.Decimal `json:"fee"`
	Volume         decimal.Decimal `json:"volume"`
	OpenTime       float64         `json:"open_time"`
	CloseTime      float64         `json:"close_time"`
}
