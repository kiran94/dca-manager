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
	envS3Bucket     string = "DCA_BUCKET"
	envS3ConfigPath string = "DCA_CONFIG"
)

type DCAConfig struct {
	Orders []DCAOrder `json:"orders"`
}

type DCAOrder struct {
	Direction string `json:"direction"`
	OrderType string `json:"ordertype"`
	Volume    string `json:"volume"`
	Pair      string `json:"pair"`
	Validate  bool   `json:"validate"`
	Enabled   bool   `json:"enabled"`
}

// Gets DCA Configuration from S3
func GetDCAConfiguration(config aws.Config, c context.Context) (*DCAConfig, error) {

	s3Bucket := os.Getenv(envS3Bucket)
	s3ConfigPath := os.Getenv(envS3ConfigPath)
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
