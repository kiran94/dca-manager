package configuration

import (
	"context"
	"encoding/json"
	"io/ioutil"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	EnvS3Bucket                        string = "DCA_BUCKET"
	EnvS3ConfigPath                    string = "DCA_CONFIG"
	EnvAllowReal                       string = "DCA_ALLOW_REAL"
	EnvSQSPendingOrdersQueue           string = "DCA_PENDING_ORDERS_QUEUE_URL"
	EnvS3PendingTransaction            string = "DCA_PENDING_ORDER_S3_PREFIX"
	EnvS3ProcessedTransaction          string = "DCA_PROCESSED_ORDER_S3_PREFIX"
	EnvGlueProcessTransactionJob       string = "DCA_GLUE_PROCESS_TRANSACTION_JOB"
	EnvGlueProcessTransactionOperation string = "DCA_GLUE_PROCESS_TRANSACTION_OPERATION"
)

// The root object for DCA configuration
type DCAConfig struct {
	Orders []DCAOrder `json:"orders"`
}

// A single order to be executed by DCA
type DCAOrder struct {
	Exchange  string `json:"exchange"`
	Direction string `json:"direction"`
	OrderType string `json:"ordertype"`
	Volume    string `json:"volume"`
	Pair      string `json:"pair"`
	Validate  bool   `json:"validate"`
	Enabled   bool   `json:"enabled"`
}

type S3Access interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

type S3 struct {
	Client *s3.Client
}

func (s *S3) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return s.Client.GetObject(ctx, params, optFns...)
}

// Gets DCA Configuration from S3
func GetDCAConfiguration(ctx context.Context, s S3Access, s3Bucket *string, s3ConfigPath *string) (*DCAConfig, error) {

	configObject, err := s.GetObject(ctx, &s3.GetObjectInput{
		Bucket: s3Bucket,
		Key:    s3ConfigPath,
	})

	if err != nil {
		return nil, err
	}

	configObjectBytes, err := ioutil.ReadAll(configObject.Body)
	if err != nil {
		return nil, err
	}
	defer configObject.Body.Close()

	var dcaConfig DCAConfig
	jsonErr := json.Unmarshal(configObjectBytes, &dcaConfig)

	if jsonErr != nil {
		return nil, jsonErr
	}

	return &dcaConfig, nil
}
