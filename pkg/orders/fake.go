package orders

import (
	krakenapi "github.com/beldur/kraken-go-api-client"
	"github.com/sirupsen/logrus"
)

// Generate a Fake Executed Order.
//
// Useful for end-to-end testing so we
// don't need to keep paying money during testing ;)
func GetFakeOrderFufilled() (*OrderFufilled, error) {
	logrus.Warn(`USING FAKE DATA. In order to execute real transactions enable the DCA_ALLOW_REAL environment variable.`)

	var orderErr error
	orderResult := &OrderFufilled{
		Result: &krakenapi.AddOrderResponse{
			TransactionIds: []string{"TXID"},
			Description: krakenapi.OrderDescription{
				AssetPair:      "ADAGBP",
				Close:          "100",
				Leverage:       "Leverage",
				Order:          "buy 10.00000000 ADAGBP @ market",
				OrderType:      "OrderType",
				PrimaryPrice:   "PrimaryPrice",
				SecondaryPrice: "SecondaryPrice",
				Type:           "Type",
			},
		},
		Timestamp:     12345678,
		TransactionId: "OEBG2U-KIRAN-4U6WHJ",
	}

	return orderResult, orderErr
}
