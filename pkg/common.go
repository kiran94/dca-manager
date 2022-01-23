package pkg

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/glue"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// AWS S3

// S3Access is an abstraction for S3 Operations
type S3Access interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

// S3 is a Concrete Wrapper for S3 Operations
type S3 struct {
	Client *s3.Client
}

// GetObject gets an object from S3
func (s S3) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return s.Client.GetObject(ctx, params, optFns...)
}

// PutObject puts an object into S3
func (s S3) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return s.Client.PutObject(ctx, params, optFns...)
}

// AWS SSM

// SSMAccess is an abstraction for SSM Operations
type SSMAccess interface {
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

// SSM is a Concrete Wrapper for SSM
type SSM struct {
	Client *ssm.Client
}

// GetParameter gets a parameter from the SSM store.
func (s SSM) GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	return s.Client.GetParameter(ctx, params, optFns...)
}

// AWS SQS

// SQSAccess is an abstraction for SQS Operations
type SQSAccess interface {
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
	DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
}

// SQS is a Concrete Wrapper for SQS
type SQS struct {
	Client *sqs.Client
}

// SendMessage sends a message to SQS
func (s SQS) SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	return s.Client.SendMessage(ctx, params, optFns...)
}

// DeleteMessage deletes a message from SQS
func (s SQS) DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	return s.Client.DeleteMessage(ctx, params, optFns...)
}

// AWS Glue

// GlueAccess is an abstraction for AWS Glue
type GlueAccess interface {
	StartJobRun(ctx context.Context, params *glue.StartJobRunInput, optFns ...func(*glue.Options)) (*glue.StartJobRunOutput, error)
}

// Glue is a Concrete for Glue operations
type Glue struct {
	Client *glue.Client
}

// StartJobRun starts a new AWS Glue job.
func (g Glue) StartJobRun(ctx context.Context, params *glue.StartJobRunInput, optFns ...func(*glue.Options)) (*glue.StartJobRunOutput, error) {
	return g.Client.StartJobRun(ctx, params, optFns...)
}
