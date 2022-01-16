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
)

type MockS3Access struct {
	mock.Mock
}

func (s MockS3Access) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	args := s.Called(ctx, params, optFns)
	return args.Get(0).(*s3.GetObjectOutput), args.Error(1)
}

// Ensures when the object cannot be found
// then an err is returned
func TestGetDCAConfigurationErrorGettingConfig(t *testing.T) {
	s3Access := MockS3Access{}
	s3Bucket := "myBucket"
	s3ConfigPath := "my/config.json"

	var o *s3.GetObjectOutput = nil
	s3Access.On("GetObject", mock.Anything, &s3.GetObjectInput{Bucket: &s3Bucket, Key: &s3ConfigPath}, mock.Anything).Return(o, errors.New("config not found")).Once()

	resultConfig, err := GetDCAConfiguration(context.Background(), s3Access, &s3Bucket, &s3ConfigPath)

	s3Access.AssertExpectations(t)
	assert.Nil(t, resultConfig)
	assert.NotNil(t, err)

	assert.Equal(t, "config not found", err.Error())
}

// Ensures when the object cannot be derserialised
// an error is raised
func TestGetDCAConfigurationCouldNotUnmarshalJson(t *testing.T) {
	s3Access := MockS3Access{}
	s3Bucket := "myBucket"
	s3ConfigPath := "my/config.json"

	r := bytes.NewReader([]byte("fake json"))
	bodyCloser := io.NopCloser(r)

	o := &s3.GetObjectOutput{Body: bodyCloser}
	s3Access.On("GetObject", mock.Anything, &s3.GetObjectInput{Bucket: &s3Bucket, Key: &s3ConfigPath}, mock.Anything).Return(o, nil).Once()

	resultConfig, err := GetDCAConfiguration(context.Background(), s3Access, &s3Bucket, &s3ConfigPath)

	s3Access.AssertExpectations(t)
	assert.Nil(t, resultConfig)
	assert.NotNil(t, err)

	assert.Contains(t, err.Error(), "invalid character")
}

// Ensures whent the configuration
// can be retrieved and deserialised
// it is returned
func TestGetDCAConfiguration(t *testing.T) {
	s3Access := MockS3Access{}
	s3Bucket := "myBucket"
	s3ConfigPath := "my/config.json"

	dcaConfig := &DCAConfig{Orders: []DCAOrder{{Exchange: "kraken"}}}
	b, _ := json.Marshal(dcaConfig)

	r := bytes.NewReader(b)
	bodyCloser := io.NopCloser(r)

	o := &s3.GetObjectOutput{Body: bodyCloser}
	s3Access.On("GetObject", mock.Anything, &s3.GetObjectInput{Bucket: &s3Bucket, Key: &s3ConfigPath}, mock.Anything).Return(o, nil).Once()

	resultConfig, err := GetDCAConfiguration(context.Background(), s3Access, &s3Bucket, &s3ConfigPath)

	assert.NotNil(t, resultConfig)
	assert.Nil(t, err)
	assert.Equal(t, dcaConfig, resultConfig)
}
