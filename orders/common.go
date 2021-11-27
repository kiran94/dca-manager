package orders

import (
	config "github.com/kiran94/dca-manager/configuration"
)

// Responsible for making DCA orders to an Exchange.
type Orderer interface {
	MakeOrder(order *config.DCAOrder) (*OrderFufilled, error)
}

// An Order which has been sent to the Exchange
type OrderFufilled struct {
	TransactionId string      `json:"transaction_id"`
	Timestamp     int64       `json:"timestamp"`
	Result        interface{} `json:"result"`
}

// A Pending Order which is processing on the exchange
// where the s3 bucket & key define the result from the
// exchange from the initial call.
//
// An order could be accepted by the exchange but not nessasarily
// successful yet so the payload in s3 is whatever they sent back to us.
//
// This object is used to push the trasnaction to an out-of-process
// queue for later processing
type PendingOrders struct {
	TransactionId string `json:"transaction_id"`
	S3Bucket      string `json:"s3_bucket"`
	S3Key         string `json:"s3_key"`
}
