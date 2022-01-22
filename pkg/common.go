package pkg

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/glue"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// AWS S3

type S3Access interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

type S3 struct {
	Client *s3.Client
}

func (s S3) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return s.Client.GetObject(ctx, params, optFns...)
}

func (s S3) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return s.Client.PutObject(ctx, params, optFns...)
}

// AWS SSM

type SSMAccess interface {
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

type SSM struct {
	Client *ssm.Client
}

func (s SSM) GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	return s.Client.GetParameter(ctx, params, optFns...)
}

// AWS SQS

type SQSAccess interface {
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
	DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
}

type SQS struct {
	Client *sqs.Client
}

func (s SQS) SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	return s.Client.SendMessage(ctx, params, optFns...)
}

func (s SQS) DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	return s.Client.DeleteMessage(ctx, params, optFns...)
}

// AWS Glue

type GlueAccess struct {
	Client *glue.Client
}

func (g GlueAccess) StartJobRun(ctx context.Context, params *glue.StartJobRunInput, optFns ...func(*glue.Options)) (*glue.StartJobRunOutput, error) {
	return g.Client.StartJobRun(ctx, params, optFns...)
}
