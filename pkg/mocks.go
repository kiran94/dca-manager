package pkg

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/glue"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/stretchr/testify/mock"
)

// MockS3Access mocks S3 operations
type MockS3Access struct {
	mock.Mock
}

// GetObject mocks getting an object from S3
func (s MockS3Access) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	args := s.Called(ctx, params, optFns)
	return args.Get(0).(*s3.GetObjectOutput), args.Error(1)
}

// PutObject mocks putting an object to s3
func (s MockS3Access) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	args := s.Called(ctx, params, optFns)
	return args.Get(0).(*s3.PutObjectOutput), args.Error(1)
}

// MockSSMClient mocks SSM operations
type MockSSMClient struct {
	mock.Mock
}

// GetParameter mocks getting a parameter from SSM.
func (s MockSSMClient) GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	args := s.Called(ctx, params, optFns)
	return args.Get(0).(*ssm.GetParameterOutput), args.Error(1)
}

// MockSQSAccess mocks SQS operations
type MockSQSAccess struct {
	mock.Mock
}

// SendMessage mocks sending a message to SQS.
func (s MockSQSAccess) SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	args := s.Called(ctx, params, optFns)
	return args.Get(0).(*sqs.SendMessageOutput), args.Error(1)
}

// DeleteMessage mocks deleting a message to SQS.
func (s MockSQSAccess) DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	args := s.Called(ctx, params, optFns)
	return args.Get(0).(*sqs.DeleteMessageOutput), args.Error(1)
}

// MockGlueAccess mocks aws glue operations
type MockGlueAccess struct {
	mock.Mock
}

// StartJobRun mocks start a glue job.
func (g MockGlueAccess) StartJobRun(ctx context.Context, params *glue.StartJobRunInput, optFns ...func(*glue.Options)) (*glue.StartJobRunOutput, error) {
	args := g.Called(ctx, params, optFns)
	return args.Get(0).(*glue.StartJobRunOutput), args.Error(1)
}
