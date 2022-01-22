package orders

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/kiran94/dca-manager/pkg"
	"github.com/sirupsen/logrus"
)

type PendingOrderQueue interface {
	SubmitPendingOrder(ctx context.Context, sc pkg.SQSAccess, po *PendingOrders, exchange string, real bool, sqsQueue string) error
}

// Submits the given pending order to queue.
type PendingOrderSubmitter struct{}

func (p PendingOrderSubmitter) SubmitPendingOrder(ctx context.Context, sc pkg.SQSAccess, po *PendingOrders, exchange string, real bool, sqsQueue string) error {
	sqsMessageBodyBytes, err := json.Marshal(po)
	if err != nil {
		return err
	}

	sqsMessage := string(sqsMessageBodyBytes)
	sqsMessageInput := &sqs.SendMessageInput{
		QueueUrl:    &sqsQueue,
		MessageBody: aws.String(sqsMessage),
		MessageAttributes: map[string]types.MessageAttributeValue{
			"Exchange": {
				DataType:    aws.String("String"),
				StringValue: aws.String(exchange),
			},
			"TransactionId": {
				DataType:    aws.String("String"),
				StringValue: aws.String(po.TransactionId),
			},
			"Real": {
				DataType:    aws.String("String"),
				StringValue: aws.String(strconv.FormatBool(real)),
			},
		},
	}

	logrus.WithFields(logrus.Fields{
		"transactionId": po.TransactionId,
		"queue":         sqsQueue,
		"real":          real,
		"exchange":      exchange,
	}).Info("Submitting Transaction to Queue")

	_, sqsErr := sc.SendMessage(ctx, sqsMessageInput)

	if sqsErr != nil {
		return sqsErr
	}

	return nil
}
