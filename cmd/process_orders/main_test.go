package main

import (
	"context"
	"errors"
	"testing"

	awsEvents "github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/glue"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/kiran94/dca-manager/pkg"
	"github.com/kiran94/dca-manager/pkg/configuration"
	"github.com/kiran94/dca-manager/pkg/orders"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Orderer Factory
type MockOrdererFactory struct {
	mock.Mock
}

func (m MockOrdererFactory) GetOrderers(ctx context.Context, ssm pkg.SSMAccess) (*map[string]orders.Orderer, error) {
	args := m.Called(ctx, ssm)
	return args.Get(0).(*map[string]orders.Orderer), args.Error(1)
}

// Kraken Orderer
type MockKrakenOrderer struct {
	mock.Mock
}

func (m MockKrakenOrderer) MakeOrder(order *configuration.DCAOrder) (*orders.OrderFufilled, error) {
	args := m.Called(order)
	return args.Get(0).(*orders.OrderFufilled), args.Error(1)
}

func (m MockKrakenOrderer) ProcessTransaction(transactionsIds ...string) (*[]orders.OrderComplete, error) {
	args := m.Called(transactionsIds)
	return args.Get(0).(*[]orders.OrderComplete), args.Error(1)
}

// Ensures when no records are found, then
// an error is returned
func TestProcessTransactionsNoRecords(t *testing.T) {
	services := &DCAServices{}
	config := &AppConfig{}
	sqsEvent := awsEvents.SQSEvent{Records: []awsEvents.SQSMessage{}}

	err := ProcessTransactions(context.Background(), services, config, sqsEvent)
	assert.Equal(t, "no sqs messages found, returning", err.Error())
}

// Ensures when there is an error getting orderers
// it is returned
func TestProcessTransactionsErrorOrderers(t *testing.T) {
	var expectedOrderer *map[string]orders.Orderer
	var expectedErr error = errors.New("error getting orderer")

	mockSsm := pkg.MockSSMClient{}
	mockOrderer := MockOrdererFactory{}
	mockOrderer.On("GetOrderers", mock.Anything, mockSsm).Return(expectedOrderer, expectedErr)

	services := &DCAServices{
		ssmAccess:      mockSsm,
		ordererFactory: mockOrderer,
	}
	config := &AppConfig{}
	sqsEvent := awsEvents.SQSEvent{Records: []awsEvents.SQSMessage{{MessageId: "ID"}}}

	err := ProcessTransactions(context.Background(), services, config, sqsEvent)
	assert.Equal(t, expectedErr, err)
}

// Ensures when the transaction is not real
// it is deleted from the queue
// and no glue job/s3 upload is run
func TestProcessTransactionsNotReal(t *testing.T) {
	queueURL := "EventSourceARN"
	recieptHandle := "recieptHandle"
	exchange := "kraken"
	isReal := "false"

	mockKrakenOrderer := MockKrakenOrderer{}
	mockKrakenOrderer.On("ProcessTransaction", "TXID").Return()
	expectedOrderer := &map[string]orders.Orderer{"kraken": mockKrakenOrderer}
	var expectedErr error

	mockSsm := pkg.MockSSMClient{}
	mockOrderer := MockOrdererFactory{}
	mockOrderer.On("GetOrderers", mock.Anything, mockSsm).Return(expectedOrderer, expectedErr)

	mockS3 := pkg.MockS3Access{}
	mockGlue := pkg.MockGlueAccess{}

	mockSqs := pkg.MockSQSAccess{}
	mockSqs.On("DeleteMessage", mock.Anything, mock.MatchedBy(func(s *sqs.DeleteMessageInput) bool {
		return (*s.QueueUrl == queueURL) && (*s.ReceiptHandle == recieptHandle)
	}), mock.Anything).Return(&sqs.DeleteMessageOutput{}, nil)

	services := &DCAServices{
		ssmAccess:      mockSsm,
		ordererFactory: mockOrderer,
		sqsAccess:      mockSqs,
		s3Access:       mockS3,
		glueAccess:     mockGlue,
	}
	config := &AppConfig{}

	sqsEvent := awsEvents.SQSEvent{
		Records: []awsEvents.SQSMessage{
			{
				MessageId:      "ID",
				ReceiptHandle:  recieptHandle,
				EventSourceARN: queueURL,
				MessageAttributes: map[string]awsEvents.SQSMessageAttribute{
					"Exchange": {StringValue: &exchange},
					"Real":     {StringValue: &isReal},
				},
				Body: `{ "transaction_id": "TXID", "s3_bucket": "bucket", "s3_key": "key" }`,
			},
		},
	}

	err := ProcessTransactions(context.Background(), services, config, sqsEvent)
	assert.Nil(t, err)

	mockSqs.AssertExpectations(t)
	mockS3.AssertNotCalled(t, "PutObject", mock.Anything, mock.Anything, mock.Anything)
	mockGlue.AssertNotCalled(t, "StartJobRun", mock.Anything, mock.Anything, mock.Anything)
}

// Ensures when the incoming transaction's exchange
// has not been confgiured then an error is raised
func TestProcessTransactionsExchangeNotFound(t *testing.T) {
	type testCase struct {
		exchange    string
		expectedErr string
	}

	cases := []testCase{
		{exchange: "binance", expectedErr: "exchange binance was not configured"},
		{exchange: "", expectedErr: "received sqs message with no exchange"},
	}

	for _, currentCase := range cases {

		queueURL := "EventSourceARN"
		recieptHandle := "recieptHandle"
		exchange := currentCase.exchange
		isReal := "true"

		sqsEvent := awsEvents.SQSEvent{
			Records: []awsEvents.SQSMessage{
				{
					MessageId:      "ID",
					ReceiptHandle:  recieptHandle,
					EventSourceARN: queueURL,
					MessageAttributes: map[string]awsEvents.SQSMessageAttribute{
						"Exchange": {StringValue: &exchange},
						"Real":     {StringValue: &isReal},
					},
					Body: `{ "transaction_id": "TXID", "s3_bucket": "bucket", "s3_key": "key" }`,
				},
			},
		}

		mockKrakenOrderer := MockKrakenOrderer{}
		mockKrakenOrderer.On("ProcessTransaction", "TXID").Return()
		expectedOrderer := &map[string]orders.Orderer{"kraken": mockKrakenOrderer}
		var expectedErr error

		mockSsm := pkg.MockSSMClient{}
		mockOrderer := MockOrdererFactory{}
		mockOrderer.On("GetOrderers", mock.Anything, mockSsm).Return(expectedOrderer, expectedErr)

		services := &DCAServices{
			ssmAccess:      mockSsm,
			ordererFactory: mockOrderer,
		}
		config := &AppConfig{}

		err := ProcessTransactions(context.Background(), services, config, sqsEvent)
		assert.Contains(t, err.Error(), currentCase.expectedErr)
	}
}

// Ensures when the transaction can
// be processed it is uploaded to s3 and glue is submitted
func TestProcessTransactions(t *testing.T) {

	bucket := "bucket"
	prefixPath := "path"
	glueJobName := "glue_job"
	queueURL := "EventSourceARN"
	recieptHandle := "recieptHandle"
	exchange := "kraken"
	isReal := "true"

	sqsEvent := awsEvents.SQSEvent{
		Records: []awsEvents.SQSMessage{
			{
				MessageId:      "ID",
				ReceiptHandle:  recieptHandle,
				EventSourceARN: queueURL,
				MessageAttributes: map[string]awsEvents.SQSMessageAttribute{
					"Exchange": {StringValue: &exchange},
					"Real":     {StringValue: &isReal},
				},
				Body: `{ "transaction_id": "TXID", "s3_bucket": "bucket", "s3_key": "key" }`,
			},
		},
	}

	mockKrakenOrderer := MockKrakenOrderer{}
	mockKrakenOrderer.On("ProcessTransaction", []string{"TXID"}).Return(&[]orders.OrderComplete{
		{
			TransactionID: "TXID",
		},
	}, nil)
	expectedOrderer := &map[string]orders.Orderer{"kraken": mockKrakenOrderer}
	var expectedErr error

	mockSsm := pkg.MockSSMClient{}
	mockOrderer := MockOrdererFactory{}
	mockOrderer.On("GetOrderers", mock.Anything, mockSsm).Return(expectedOrderer, expectedErr)

	mockS3 := pkg.MockS3Access{}
	mockS3.On("PutObject", mock.Anything, mock.Anything, mock.Anything).Return(&s3.PutObjectOutput{}, nil)

	mockGlue := pkg.MockGlueAccess{}
	jobID := "jobId"
	mockGlue.On("StartJobRun", mock.Anything, mock.Anything, mock.Anything).Return(&glue.StartJobRunOutput{JobRunId: &jobID}, nil)

	mockSqs := pkg.MockSQSAccess{}
	mockSqs.On("DeleteMessage", mock.Anything, mock.Anything, mock.Anything).Return(&sqs.DeleteMessageOutput{}, nil)

	services := &DCAServices{
		ssmAccess:      mockSsm,
		ordererFactory: mockOrderer,
		s3Access:       mockS3,
		glueAccess:     mockGlue,
		sqsAccess:      mockSqs,
	}
	config := AppConfig{s3bucket: bucket, dcaConfigPath: prefixPath}
	config.glue.processTransactionJob = glueJobName
	config.glue.processTransactionOperation = "upsert"

	err := ProcessTransactions(context.Background(), services, &config, sqsEvent)
	assert.Nil(t, err)
}
