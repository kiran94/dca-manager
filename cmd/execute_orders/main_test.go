package main

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/kiran94/dca-manager/pkg"
	"github.com/kiran94/dca-manager/pkg/configuration"
	"github.com/kiran94/dca-manager/pkg/orders"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Kraken Orderer
type MockKrakenOrderer struct {
	mock.Mock
}

func (m *MockKrakenOrderer) MakeOrder(order *configuration.DCAOrder) (*orders.OrderFufilled, error) {
	args := m.Called(order)
	return args.Get(0).(*orders.OrderFufilled), args.Error(1)
}

func (m *MockKrakenOrderer) ProcessTransaction(transactionsIds ...string) (*[]orders.OrderComplete, error) {
	args := m.Called(transactionsIds)
	return args.Get(0).(*[]orders.OrderComplete), args.Error(1)
}

// DCA Configration
type MockDCAConfiguration struct {
	mock.Mock
}

func (d MockDCAConfiguration) GetDCAConfiguration(ctx context.Context, s3Client pkg.S3Access, s3Bucket *string, s3ConfigPath *string) (*configuration.DCAConfig, error) {
	args := d.Called(ctx, s3Client, s3Bucket, s3ConfigPath)
	return args.Get(0).(*configuration.DCAConfig), args.Error(1)
}

// Orderer Factory
type MockOrdererFactory struct {
	mock.Mock
}

func (m MockOrdererFactory) GetOrderers(ctx context.Context, ssm pkg.SSMAccess) (*map[string]orders.Orderer, error) {
	args := m.Called(ctx, ssm)
	return args.Get(0).(*map[string]orders.Orderer), args.Error(1)
}

// Pending Order Submitter
type MockPendingOrderSubmitter struct {
	mock.Mock
}

func (m MockPendingOrderSubmitter) SubmitPendingOrder(ctx context.Context, sc pkg.SQSAccess, po *orders.PendingOrders, exchange string, real bool, sqsQueue string) error {
	args := m.Called(ctx, sc, po, exchange, real, sqsQueue)
	return args.Error(0)
}

/*
   Generates a fresh Services and Configuration
   Expectations are set via incoming func
*/
func setup(apply func(s3 *pkg.MockS3Access, ssm *pkg.MockSSMClient, sqs *pkg.MockSQSAccess, c *MockDCAConfiguration, o *MockOrdererFactory, po *MockPendingOrderSubmitter, appConfig *AppConfig)) (*DCAServices, *AppConfig) {

	appConfig := &AppConfig{s3bucket: "bucket", dcaConfigPath: "config_path", allowReal: false}
	appConfig.transactions.pendingS3TransactionPrefix = "s3_pending_prefix"
	appConfig.transactions.processedS3TransactionPrefix = "s3_processed_prefix"
	appConfig.queue.sqsURL = "sqs_url"
	appConfig.glue.processTransactionJob = "process_transaction_glue_job"
	appConfig.glue.processTransactionOperation = "process_transaction_glue_operation"

	awsConfig := aws.Config{}
	s3Access := &pkg.MockS3Access{}
	ssmAccess := &pkg.MockSSMClient{}
	sqsAccess := &pkg.MockSQSAccess{}
	configSource := &MockDCAConfiguration{}
	ordererFactory := &MockOrdererFactory{}
	pendingOrderSubmitter := &MockPendingOrderSubmitter{}
	apply(s3Access, ssmAccess, sqsAccess, configSource, ordererFactory, pendingOrderSubmitter, appConfig)

	services := &DCAServices{
		awsConfig:             awsConfig,
		s3Access:              s3Access,
		ssmAccess:             ssmAccess,
		sqsAccess:             sqsAccess,
		configSource:          configSource,
		ordererFactory:        ordererFactory,
		pendingOrderSubmitter: pendingOrderSubmitter,
	}

	return services, appConfig
}

func AssertExpectations(t *testing.T, services *DCAServices) {
	services.s3Access.(*pkg.MockS3Access).AssertExpectations(t)
	services.ssmAccess.(*pkg.MockSSMClient).AssertExpectations(t)
	services.sqsAccess.(*pkg.MockSQSAccess).AssertExpectations(t)
	services.configSource.(*MockDCAConfiguration).AssertExpectations(t)
	services.ordererFactory.(*MockOrdererFactory).AssertExpectations(t)
	services.pendingOrderSubmitter.(*MockPendingOrderSubmitter).AssertExpectations(t)
}

// Ensures when an error is returned when getting the DCA config
// an error is returned
func TestExecuteOrdersErrorGettingConfig(t *testing.T) {
	var expectedErr error = errors.New("error getting config")
	services, appConfig := setup(func(s3 *pkg.MockS3Access, ssm *pkg.MockSSMClient, sqs *pkg.MockSQSAccess, c *MockDCAConfiguration, o *MockOrdererFactory, po *MockPendingOrderSubmitter, appConfig *AppConfig) {
		c.On("GetDCAConfiguration", mock.Anything, s3, &appConfig.s3bucket, &appConfig.dcaConfigPath).Return(&configuration.DCAConfig{}, expectedErr)
	})

	pos, err := ExecuteOrders(context.Background(), services, appConfig)
	assert.Nil(t, pos)
	assert.Equal(t, expectedErr, err)

	AssertExpectations(t, services)
}

// Ensures when an error is returned when getting
// Orderers an error is returned
func TestExecuteOrdersErrorGettingOrderers(t *testing.T) {
	expectedConfig := &configuration.DCAConfig{}

	expectedOrdererResult := &map[string]orders.Orderer{}
	var expectedOrdererErr error = errors.New("error getting orderer")

	services, appConfig := setup(func(s3 *pkg.MockS3Access, ssm *pkg.MockSSMClient, sqs *pkg.MockSQSAccess, c *MockDCAConfiguration, o *MockOrdererFactory, po *MockPendingOrderSubmitter, appConfig *AppConfig) {
		c.On("GetDCAConfiguration", mock.Anything, s3, &appConfig.s3bucket, &appConfig.dcaConfigPath).Return(expectedConfig, nil)
		o.On("GetOrderers", mock.Anything, mock.Anything).Return(expectedOrdererResult, expectedOrdererErr)
	})

	pos, err := ExecuteOrders(context.Background(), services, appConfig)
	assert.Nil(t, pos)
	assert.Equal(t, expectedOrdererErr, err)

	AssertExpectations(t, services)
}

// Ensures when no orders are configured,
// empty pending are returned
func TestExecuteOrdersNoConfiguredOrders(t *testing.T) {
	dcaConfig := &configuration.DCAConfig{Orders: []configuration.DCAOrder{}}
	mockOrderer := &MockKrakenOrderer{}
	expectedOrdererResult := &map[string]orders.Orderer{"kraken": mockOrderer}

	services, appConfig := setup(func(s3 *pkg.MockS3Access, ssm *pkg.MockSSMClient, sqs *pkg.MockSQSAccess, c *MockDCAConfiguration, o *MockOrdererFactory, po *MockPendingOrderSubmitter, appConfig *AppConfig) {
		c.On("GetDCAConfiguration", mock.Anything, s3, &appConfig.s3bucket, &appConfig.dcaConfigPath).Return(dcaConfig, nil)
		o.On("GetOrderers", mock.Anything, mock.Anything).Return(expectedOrdererResult, nil)

	})
	mockOrderer.On("MakeOrder", mock.Anything).Times(0)

	pos, err := ExecuteOrders(context.Background(), services, appConfig)

	assert.Equal(t, 0, len(*pos))
	assert.Equal(t, 0, cap(*pos))
	assert.Nil(t, err)

	AssertExpectations(t, services)
	services.s3Access.(*pkg.MockS3Access).AssertNotCalled(t, "PutObject", mock.Anything, mock.Anything, mock.Anything)
	services.pendingOrderSubmitter.(*MockPendingOrderSubmitter).AssertNotCalled(t, "SubmitPendingOrder")
}

// Ensures when no orderer is configured
// for the order exchange
// an error is returned
func TestExecuteOrdersNoOrdererForExchange(t *testing.T) {
	dcaConfig := &configuration.DCAConfig{Orders: []configuration.DCAOrder{
		{
			Exchange:  "binance",
			Pair:      "BTCGBP",
			Volume:    "1",
			Direction: "buy",
		},
	}}
	mockOrderer := &MockKrakenOrderer{}
	expectedOrdererResult := &map[string]orders.Orderer{"kraken": mockOrderer}

	services, appConfig := setup(func(s3 *pkg.MockS3Access, ssm *pkg.MockSSMClient, sqs *pkg.MockSQSAccess, c *MockDCAConfiguration, o *MockOrdererFactory, po *MockPendingOrderSubmitter, appConfig *AppConfig) {
		c.On("GetDCAConfiguration", mock.Anything, s3, &appConfig.s3bucket, &appConfig.dcaConfigPath).Return(dcaConfig, nil)
		o.On("GetOrderers", mock.Anything, mock.Anything).Return(expectedOrdererResult, nil)

	})
	mockOrderer.On("MakeOrder", mock.Anything).Times(0)

	appConfig.allowReal = true
	pos, err := ExecuteOrders(context.Background(), services, appConfig)

	assert.Nil(t, pos)
	assert.Contains(t, err.Error(), "no orderer found for exchange binance")
}

// Ensures when allo real is disabled, then
// the fake transaction is used
func TestExecuteOrdersAllowRealDisabled(t *testing.T) {
	dcaConfig := &configuration.DCAConfig{Orders: []configuration.DCAOrder{
		{
			Exchange:  "kraken",
			Pair:      "BTCGBP",
			Volume:    "1",
			Direction: "buy",
		},
	}}
	mockOrderer := &MockKrakenOrderer{}
	expectedOrdererResult := &map[string]orders.Orderer{"kraken": mockOrderer}
	expectedS3PutObject := &s3.PutObjectOutput{}
	var expectedErr error

	services, appConfig := setup(func(s3 *pkg.MockS3Access, ssm *pkg.MockSSMClient, sqs *pkg.MockSQSAccess, c *MockDCAConfiguration, o *MockOrdererFactory, po *MockPendingOrderSubmitter, appConfig *AppConfig) {
		c.On("GetDCAConfiguration", mock.Anything, s3, &appConfig.s3bucket, &appConfig.dcaConfigPath).Return(dcaConfig, nil)
		o.On("GetOrderers", mock.Anything, mock.Anything).Return(expectedOrdererResult, nil)
		s3.On("PutObject", mock.Anything, mock.Anything, mock.Anything).Return(expectedS3PutObject, nil)
		po.On("SubmitPendingOrder", mock.Anything, sqs, mock.Anything, "kraken", appConfig.allowReal, appConfig.queue.sqsURL).Return(expectedErr)
	})
	mockOrderer.On("MakeOrder", mock.Anything).Times(0)

	appConfig.allowReal = false
	pos, err := ExecuteOrders(context.Background(), services, appConfig)

	assert.NotNil(t, pos)
	assert.Nil(t, err)

	AssertExpectations(t, services)
	assert.Equal(t, 1, len(*pos))
	assert.Equal(t, "bucket", (*pos)[0].S3Bucket)
	assert.Equal(t, "s3_pending_prefix/exchange=kraken/OEBG2U-KIRAN-4U6WHJ.json", (*pos)[0].S3Key)
	assert.Equal(t, "OEBG2U-KIRAN-4U6WHJ", (*pos)[0].TransactionID)
}

// Ensures when there is an error
// uploading the object
// it is propagated
func TestExecuteOrdersErrorUploadingObject(t *testing.T) {
	dcaConfig := &configuration.DCAConfig{Orders: []configuration.DCAOrder{
		{
			Exchange:  "kraken",
			Pair:      "BTCGBP",
			Volume:    "1",
			Direction: "buy",
		},
	}}
	mockOrderer := &MockKrakenOrderer{}
	expectedOrdererResult := &map[string]orders.Orderer{"kraken": mockOrderer}
	expectedS3PutObject := &s3.PutObjectOutput{}
	var expectedErr error = errors.New("error uploading object")

	services, appConfig := setup(func(s3 *pkg.MockS3Access, ssm *pkg.MockSSMClient, sqs *pkg.MockSQSAccess, c *MockDCAConfiguration, o *MockOrdererFactory, po *MockPendingOrderSubmitter, appConfig *AppConfig) {
		c.On("GetDCAConfiguration", mock.Anything, s3, &appConfig.s3bucket, &appConfig.dcaConfigPath).Return(dcaConfig, nil)
		o.On("GetOrderers", mock.Anything, mock.Anything).Return(expectedOrdererResult, nil)
		s3.On("PutObject", mock.Anything, mock.Anything, mock.Anything).Return(expectedS3PutObject, expectedErr)
		po.On("SubmitPendingOrder", mock.Anything, sqs, mock.Anything, "kraken", appConfig.allowReal, appConfig.queue.sqsURL).Return(nil)
	})
	mockOrderer.On("MakeOrder", mock.Anything).Times(0)

	appConfig.allowReal = false
	pos, err := ExecuteOrders(context.Background(), services, appConfig)

	assert.Nil(t, pos)
	assert.Equal(t, expectedErr, err)
}

// Ensures when there is an error submitting pending orders
// an error is returned
func TestExecuteOrdersErrorSubmittingQueue(t *testing.T) {
	dcaConfig := &configuration.DCAConfig{Orders: []configuration.DCAOrder{
		{
			Exchange:  "kraken",
			Pair:      "BTCGBP",
			Volume:    "1",
			Direction: "buy",
		},
	}}
	mockOrderer := &MockKrakenOrderer{}
	expectedOrdererResult := &map[string]orders.Orderer{"kraken": mockOrderer}
	expectedS3PutObject := &s3.PutObjectOutput{}
	var expectedErr error = errors.New("error submit pending order")

	services, appConfig := setup(func(s3 *pkg.MockS3Access, ssm *pkg.MockSSMClient, sqs *pkg.MockSQSAccess, c *MockDCAConfiguration, o *MockOrdererFactory, po *MockPendingOrderSubmitter, appConfig *AppConfig) {
		c.On("GetDCAConfiguration", mock.Anything, s3, &appConfig.s3bucket, &appConfig.dcaConfigPath).Return(dcaConfig, nil)
		o.On("GetOrderers", mock.Anything, mock.Anything).Return(expectedOrdererResult, nil)
		s3.On("PutObject", mock.Anything, mock.Anything, mock.Anything).Return(expectedS3PutObject, nil)
		po.On("SubmitPendingOrder", mock.Anything, sqs, mock.Anything, "kraken", appConfig.allowReal, appConfig.queue.sqsURL).Return(expectedErr)
	})
	mockOrderer.On("MakeOrder", mock.Anything).Times(0)

	appConfig.allowReal = false
	pos, err := ExecuteOrders(context.Background(), services, appConfig)

	assert.Nil(t, pos)
	assert.Equal(t, expectedErr, err)
}

// Ensures when there is an order error, the error is returned
func TestExecuteOrdersAllowRealOrderError(t *testing.T) {
	dcaConfig := &configuration.DCAConfig{Orders: []configuration.DCAOrder{
		{
			Exchange:  "kraken",
			Pair:      "BTCGBP",
			Volume:    "1",
			Direction: "buy",
		},
	}}
	mockOrderer := &MockKrakenOrderer{}
	expectedOrdererResult := &map[string]orders.Orderer{"kraken": mockOrderer}
	expectedS3PutObject := &s3.PutObjectOutput{}

	services, appConfig := setup(func(s3 *pkg.MockS3Access, ssm *pkg.MockSSMClient, sqs *pkg.MockSQSAccess, c *MockDCAConfiguration, o *MockOrdererFactory, po *MockPendingOrderSubmitter, appConfig *AppConfig) {
		c.On("GetDCAConfiguration", mock.Anything, s3, &appConfig.s3bucket, &appConfig.dcaConfigPath).Return(dcaConfig, nil)
		o.On("GetOrderers", mock.Anything, mock.Anything).Return(expectedOrdererResult, nil)
		s3.On("PutObject", mock.Anything, mock.Anything, mock.Anything).Return(expectedS3PutObject, nil)
		po.On("SubmitPendingOrder", mock.Anything, sqs, mock.Anything, "kraken", appConfig.allowReal, appConfig.queue.sqsURL).Return(nil)
	})

	var expectedOrderFufilled = &orders.OrderFufilled{}
	var expectedError error = errors.New("error making order")
	mockOrderer.On("MakeOrder", &dcaConfig.Orders[0]).Return(expectedOrderFufilled, expectedError)

	appConfig.allowReal = true
	pos, err := ExecuteOrders(context.Background(), services, appConfig)
	assert.Nil(t, pos)
	assert.NotNil(t, err)
	assert.Equal(t, expectedError, err)
}

func TestExecuteOrdersAllowRealOrderSuccessful(t *testing.T) {
	dcaConfig := &configuration.DCAConfig{Orders: []configuration.DCAOrder{
		{
			Exchange:  "kraken",
			Pair:      "BTCGBP",
			Volume:    "1",
			Direction: "buy",
		},
	}}
	mockOrderer := &MockKrakenOrderer{}
	expectedOrdererResult := &map[string]orders.Orderer{"kraken": mockOrderer}
	expectedS3PutObject := &s3.PutObjectOutput{}

	services, appConfig := setup(func(s3 *pkg.MockS3Access, ssm *pkg.MockSSMClient, sqs *pkg.MockSQSAccess, c *MockDCAConfiguration, o *MockOrdererFactory, po *MockPendingOrderSubmitter, appConfig *AppConfig) {
		appConfig.allowReal = true

		c.On("GetDCAConfiguration", mock.Anything, s3, &appConfig.s3bucket, &appConfig.dcaConfigPath).Return(dcaConfig, nil)
		o.On("GetOrderers", mock.Anything, mock.Anything).Return(expectedOrdererResult, nil)
		s3.On("PutObject", mock.Anything, mock.Anything, mock.Anything).Return(expectedS3PutObject, nil)
		po.On("SubmitPendingOrder", mock.Anything, sqs, mock.Anything, "kraken", appConfig.allowReal, appConfig.queue.sqsURL).Return(nil)
	})

	var expectedOrderFufilled = &orders.OrderFufilled{
		TransactionID: "TXID",
		Timestamp:     10002202,
	}
	mockOrderer.On("MakeOrder", &dcaConfig.Orders[0]).Return(expectedOrderFufilled, nil)

	pos, err := ExecuteOrders(context.Background(), services, appConfig)

	assert.NotNil(t, pos)
	assert.Nil(t, err)

	AssertExpectations(t, services)
	assert.Equal(t, 1, len(*pos))
	assert.Equal(t, "bucket", (*pos)[0].S3Bucket)
	assert.Equal(t, "s3_pending_prefix/exchange=kraken/TXID.json", (*pos)[0].S3Key)
	assert.Equal(t, "TXID", (*pos)[0].TransactionID)
}
