package orders

import (
	"testing"

	krakenapi "github.com/beldur/kraken-go-api-client"
	"github.com/stretchr/testify/assert"
)

// Ensures that fake data can be retrieved
func TestGetFakeOrderFufilled(t *testing.T) {
	order, err := GetFakeOrderFufilled()

	assert.NotNil(t, order)
	assert.Nil(t, err)

	assert.IsType(t, &krakenapi.AddOrderResponse{}, order.Result)
	assert.NotNil(t, order.Result.(*krakenapi.AddOrderResponse).Description)
	assert.NotEmpty(t, order.Result.(*krakenapi.AddOrderResponse).TransactionIds)
}
