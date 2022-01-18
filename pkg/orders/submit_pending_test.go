package orders

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockSQSAccess struct {
	mock.Mock
}

func (s MockSQSAccess) SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	args := s.Called(ctx, params, optFns)
	return args.Get(0).(*sqs.SendMessageOutput), args.Error(1)
}

// Ensures when there is an error submitting the message,
// it is returned
func TestSubmitPendingOrderErrorSubmittingMessage(t *testing.T) {
	mockSQS := MockSQSAccess{}
	pendingOrder := PendingOrderSubmitter{}

	var expectedSQSReturn *sqs.SendMessageOutput
	err := errors.New("error sending output")
	mockSQS.On("SendMessage", mock.Anything, mock.Anything, mock.Anything).Return(expectedSQSReturn, err)

	po := &PendingOrders{TransactionId: "TXID", S3Bucket: "bucket", S3Key: "key"}
	exchange := "exchange"
	real := false
	sqsQueue := "queue_url"

	actualErr := pendingOrder.SubmitPendingOrder(context.Background(), mockSQS, po, exchange, real, sqsQueue)
	assert.Equal(t, err, actualErr)
}

// Ensures when error is not raised, nil is returned
func TestSubmitPendingOrder(t *testing.T) {

	mockSQS := MockSQSAccess{}
	pendingOrder := PendingOrderSubmitter{}

	expectedSQSReturn := &sqs.SendMessageOutput{}
	var err error
	mockSQS.On("SendMessage", mock.Anything, mock.Anything, mock.Anything).Return(expectedSQSReturn, err)

	po := &PendingOrders{TransactionId: "TXID", S3Bucket: "bucket", S3Key: "key"}
	exchange := "exchange"
	real := false
	sqsQueue := "queue_url"

	actualErr := pendingOrder.SubmitPendingOrder(context.Background(), mockSQS, po, exchange, real, sqsQueue)
	assert.Nil(t, actualErr)
}
