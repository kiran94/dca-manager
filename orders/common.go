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

// Represents an inlight pending order
type PendingOrders struct {
	TransactionId string `json:"transaction_id"`
	S3Bucket      string `json:"s3_bucket"`
	S3Key         string `json:"s3_key"`
}
