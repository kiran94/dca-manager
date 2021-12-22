package configuration

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
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

// Gets DCA Configuration from S3
func GetDCAConfiguration(config aws.Config, c context.Context) (*DCAConfig, error) {

	s3Bucket := os.Getenv(EnvS3Bucket)
	s3ConfigPath := os.Getenv(EnvS3ConfigPath)
	s3Client := s3.NewFromConfig(config)

	configObject, err := s3Client.GetObject(c, &s3.GetObjectInput{
		Bucket: &s3Bucket,
		Key:    &s3ConfigPath,
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
