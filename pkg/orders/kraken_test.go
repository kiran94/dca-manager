package orders

import (
	"errors"
	"testing"

	krakenapi "github.com/beldur/kraken-go-api-client"
	"github.com/kiran94/dca-manager/pkg/configuration"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockKrakenAccess struct {
	mock.Mock
}

func (m *MockKrakenAccess) AddOrder(pair string, direction string, orderType string, volume string, args map[string]string) (*krakenapi.AddOrderResponse, error) {
	callArgs := m.Called(pair, direction, orderType, volume, args)
	return callArgs.Get(0).(*krakenapi.AddOrderResponse), callArgs.Error(1)
}

func (m *MockKrakenAccess) QueryOrders(txids string, args map[string]string) (*krakenapi.QueryOrdersResponse, error) {
	callArgs := m.Called(txids, args)
	return callArgs.Get(0).(*krakenapi.QueryOrdersResponse), callArgs.Error(1)
}

// Ensures when the incoming order is disabled, nothing is run
func TestMakeOrderDisabled(t *testing.T) {
	order := configuration.DCAOrder{Enabled: false}

	krakenOrder := KrakenOrderer{}
	krakenOrder.Client = &MockKrakenAccess{}
	fulfilled, err := krakenOrder.MakeOrder(&order)

	assert.Nil(t, fulfilled)
	assert.Nil(t, err)

	mockKrakenAccess := MockKrakenAccess{}
	mockKrakenAccess.AssertNotCalled(t, "AddOrder")
	mockKrakenAccess.AssertNotCalled(t, "QueryOrders")
}

// Ensures when there is an error executing
// an order, it is returned
func TestMakeOrderErrorExecutingOrder(t *testing.T) {
	order := configuration.DCAOrder{
		Enabled:   true,
		Pair:      "BTCGBP",
		Direction: "buy",
		OrderType: "market",
		Volume:    "10",
	}

	krakenOrder := KrakenOrderer{}

	var expectedAddOrderResponse *krakenapi.AddOrderResponse
	exepectedErr := errors.New("error executing order")

	m := MockKrakenAccess{}
	m.On("AddOrder", order.Pair, order.Direction, order.OrderType, order.Volume, mock.Anything).Return(expectedAddOrderResponse, exepectedErr).Once()
	krakenOrder.Client = &m

	fulfilled, err := krakenOrder.MakeOrder(&order)
	assert.Nil(t, fulfilled)
	assert.NotNil(t, err)
	assert.Equal(t, exepectedErr, err)
	m.AssertExpectations(t)
}

// Ensures when no transaction ids
// are returned then an error is returned
func TestMakeOrderNoTransactionIds(t *testing.T) {
	order := configuration.DCAOrder{
		Enabled:   true,
		Pair:      "BTCGBP",
		Direction: "buy",
		OrderType: "market",
		Volume:    "10",
	}

	krakenOrder := KrakenOrderer{}

	expectedAddOrderResponse := &krakenapi.AddOrderResponse{TransactionIds: []string{}}
	expectedErr := errors.New("No Transaction Ids recieved")

	m := MockKrakenAccess{}
	m.On("AddOrder", order.Pair, order.Direction, order.OrderType, order.Volume, mock.Anything).Return(expectedAddOrderResponse, expectedErr).Once()
	krakenOrder.Client = &m

	fulfilled, err := krakenOrder.MakeOrder(&order)

	assert.Nil(t, fulfilled)
	assert.Equal(t, expectedErr, err)
	m.AssertExpectations(t)
}

// Ensures when an order looks good
// it is wrapped and returned
func TestMakeOrder(t *testing.T) {

	order := configuration.DCAOrder{
		Enabled:   true,
		Pair:      "BTCGBP",
		Direction: "buy",
		OrderType: "market",
		Volume:    "10",
	}

	krakenOrder := KrakenOrderer{}
	expectedAddOrderResponse := &krakenapi.AddOrderResponse{
		TransactionIds: []string{"TXID"},
		Description:    krakenapi.OrderDescription{Close: "close"},
	}

	m := MockKrakenAccess{}
	m.On("AddOrder", order.Pair, order.Direction, order.OrderType, order.Volume, mock.Anything).Return(expectedAddOrderResponse, nil).Once()
	krakenOrder.Client = &m

	fulfilled, err := krakenOrder.MakeOrder(&order)

	assert.Equal(t, expectedAddOrderResponse, fulfilled.Result)
	assert.Equal(t, "TXID", fulfilled.TransactionId)
	assert.NotEqual(t, 0, fulfilled.Timestamp)
	assert.Nil(t, err)
	m.AssertExpectations(t)
}

// Ensures when no transactions are provided
// an error is returned
func TestProcessTransactionsNoTransactions(t *testing.T) {
	m := MockKrakenAccess{}
	krakenOrder := KrakenOrderer{}
	krakenOrder.Client = &m

	order, err := krakenOrder.ProcessTransaction()
	assert.Nil(t, order)
	assert.NotNil(t, err)
	assert.Contains(t, "no transactions provided", err.Error())
}

// Ensures when there is an error querying
// then the error is returned
func TestProcessTransactionsErrorQuerying(t *testing.T) {
	m := MockKrakenAccess{}
	krakenOrder := KrakenOrderer{}
	krakenOrder.Client = &m

	var expectedOrderResponse *krakenapi.QueryOrdersResponse
	var expectedErr error = errors.New("error querying")

	transactionId := "TXID"
	m.On("QueryOrders", transactionId, mock.Anything).Return(expectedOrderResponse, expectedErr)

	order, err := krakenOrder.ProcessTransaction(transactionId)

	assert.Nil(t, order)
	assert.NotNil(t, err)
	assert.Equal(t, expectedErr, err)
}

// Ensures when transaactions are returned
// they are wrapped and returned
func TestProcessTransactions(t *testing.T) {
	m := MockKrakenAccess{}
	krakenOrder := KrakenOrderer{}
	krakenOrder.Client = &m

	returnOrderResponse := &krakenapi.QueryOrdersResponse{
		"TXID": krakenapi.Order{
			TransactionID: "TXID",
			Status:        "status",
			Price:         100.23,
			Fee:           1.23,
			Volume:        "20",
			OpenTime:      2000021133,
			CloseTime:     2000021133,
			Description:   krakenapi.OrderDescription{AssetPair: "pair", OrderType: "ordertype"},
		},
	}
	var expectedErr error

	transactionId := "TXID"
	m.On("QueryOrders", transactionId, mock.Anything).Return(returnOrderResponse, expectedErr)

	orders, err := krakenOrder.ProcessTransaction(transactionId)

	assert.NotNil(t, orders)
	assert.Nil(t, err)
	m.AssertExpectations(t)

	expectedOrderResponse := OrderComplete{
		TransactionId:  (*returnOrderResponse)["TXID"].TransactionID,
		ExchangeStatus: (*returnOrderResponse)["TXID"].Status,
		Pair:           (*returnOrderResponse)["TXID"].Description.AssetPair,
		OrderType:      (*returnOrderResponse)["TXID"].Description.OrderType,
		Type:           (*returnOrderResponse)["TXID"].Description.Type,
		Price:          decimal.NewFromFloat((*returnOrderResponse)["TXID"].Price),
		Fee:            decimal.NewFromFloat((*returnOrderResponse)["TXID"].Fee),
		Volume:         decimal.NewFromFloat((*returnOrderResponse)["TXID"].VolumeExecuted),
		OpenTime:       (*returnOrderResponse)["TXID"].OpenTime,
		CloseTime:      (*returnOrderResponse)["TXID"].CloseTime,
	}

	assert.Equal(t, expectedOrderResponse, (*orders)[0])
}
