package configuration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/kiran94/dca-manager/pkg"
)


// Ensures when the object cannot be found
// then an err is returned
func TestGetDCAConfigurationErrorGettingConfig(t *testing.T) {
	s3Access := pkg.MockS3Access{}
	s3Bucket := "myBucket"
	s3ConfigPath := "my/config.json"

	var o *s3.GetObjectOutput = nil
	s3Access.On("GetObject", mock.Anything, &s3.GetObjectInput{Bucket: &s3Bucket, Key: &s3ConfigPath}, mock.Anything).Return(o, errors.New("config not found")).Once()

	dcaConfig := DCAConfiguration{}
	resultConfig, err := dcaConfig.GetDCAConfiguration(context.Background(), s3Access, &s3Bucket, &s3ConfigPath)

	s3Access.AssertExpectations(t)
	assert.Nil(t, resultConfig)
	assert.NotNil(t, err)

	assert.Equal(t, "config not found", err.Error())
}

// Ensures when the object cannot be derserialised
// an error is raised
func TestGetDCAConfigurationCouldNotUnmarshalJson(t *testing.T) {
	s3Access := pkg.MockS3Access{}
	s3Bucket := "myBucket"
	s3ConfigPath := "my/config.json"

	r := bytes.NewReader([]byte("fake json"))
	bodyCloser := io.NopCloser(r)

	o := &s3.GetObjectOutput{Body: bodyCloser}
	s3Access.On("GetObject", mock.Anything, &s3.GetObjectInput{Bucket: &s3Bucket, Key: &s3ConfigPath}, mock.Anything).Return(o, nil).Once()

	dcaConfig := DCAConfiguration{}
	resultConfig, err := dcaConfig.GetDCAConfiguration(context.Background(), s3Access, &s3Bucket, &s3ConfigPath)

	s3Access.AssertExpectations(t)
	assert.Nil(t, resultConfig)
	assert.NotNil(t, err)

	assert.Contains(t, err.Error(), "invalid character")
}

// Ensures whent the configuration
// can be retrieved and deserialised
// it is returned
func TestGetDCAConfiguration(t *testing.T) {
	s3Access := pkg.MockS3Access{}
	s3Bucket := "myBucket"
	s3ConfigPath := "my/config.json"

	dcaConfig := &DCAConfig{Orders: []DCAOrder{{Exchange: "kraken"}}}
	b, _ := json.Marshal(dcaConfig)

	r := bytes.NewReader(b)
	bodyCloser := io.NopCloser(r)

	o := &s3.GetObjectOutput{Body: bodyCloser}
	s3Access.On("GetObject", mock.Anything, &s3.GetObjectInput{Bucket: &s3Bucket, Key: &s3ConfigPath}, mock.Anything).Return(o, nil).Once()

	config := DCAConfiguration{}
	resultConfig, err := config.GetDCAConfiguration(context.Background(), s3Access, &s3Bucket, &s3ConfigPath)

	assert.NotNil(t, resultConfig)
	assert.Nil(t, err)
	assert.Equal(t, dcaConfig, resultConfig)
}
