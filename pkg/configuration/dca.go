package configuration

import (
	"context"
	"encoding/json"
	"io/ioutil"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/kiran94/dca-manager/pkg"
)

// Environment Variable names to retrieve.
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

// DCAConfig is the root object for DCA configuration.
type DCAConfig struct {
	Orders []DCAOrder `json:"orders"`
}

// DCAOrder is a single order to be executed
type DCAOrder struct {
	Exchange  string `json:"exchange"`
	Direction string `json:"direction"`
	OrderType string `json:"ordertype"`
	Volume    string `json:"volume"`
	Pair      string `json:"pair"`
	Validate  bool   `json:"validate"`
	Enabled   bool   `json:"enabled"`
}

// DCAConfiguration gets configuration from an underlying source.
type DCAConfiguration struct{}

// DCAConfigurationSource is an abstraction to get configuration.
type DCAConfigurationSource interface {
	GetDCAConfiguration(ctx context.Context, s3Client pkg.S3Access, s3Bucket *string, s3ConfigPath *string) (*DCAConfig, error)
}

// GetDCAConfiguration gets DCA configuration from S3
func (d DCAConfiguration) GetDCAConfiguration(ctx context.Context, s3Client pkg.S3Access, s3Bucket *string, s3ConfigPath *string) (*DCAConfig, error) {

	configObject, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
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

	var dcaConfig DCAConfig
	jsonErr := json.Unmarshal(configObjectBytes, &dcaConfig)

	if jsonErr != nil {
		return nil, jsonErr
	}

	return &dcaConfig, nil
}
