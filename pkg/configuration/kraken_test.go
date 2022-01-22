package configuration

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/kiran94/dca-manager/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Ensures when there is an error getting the kraken key
// an error is returned
func TestGetKrakenDetailsErrorGettingKey(t *testing.T) {
	var expectedParameter *ssm.GetParameterOutput
	expectedErr := errors.New("error getting key")

	expectedInput := &ssm.GetParameterInput{Name: &SSMKrakenKey, WithDecryption: true}
	mockSSM := pkg.MockSSMClient{}
	mockSSM.On("GetParameter", mock.Anything, expectedInput, mock.Anything).Return(expectedParameter, expectedErr)

	krakenConfig := KrakenConf{}
	key, secret, err := krakenConfig.GetKrakenDetails(context.Background(), &mockSSM)

	mockSSM.AssertExpectations(t)
	assert.Nil(t, key)
	assert.Nil(t, secret)
	assert.NotNil(t, err)

	assert.Equal(t, expectedErr, err)
}

// Ensures when there is an error getting the kraken secret
// an error is returned
func TestGetKrakenDetailsErrorGettingSecret(t *testing.T) {
	expectedKeyInput := &ssm.GetParameterInput{Name: &SSMKrakenKey, WithDecryption: true}
	expectedSecretInput := &ssm.GetParameterInput{Name: &SSMKrakenSecret, WithDecryption: true}

	expectedkey := "key"
	expectedKeyOutput := &ssm.GetParameterOutput{Parameter: &types.Parameter{Value: &expectedkey}}
	var expectedSecretOutput *ssm.GetParameterOutput
	expectedErr := errors.New("error getting key")

	mockSSM := pkg.MockSSMClient{}
	mockSSM.On("GetParameter", mock.Anything, expectedKeyInput, mock.Anything).Return(expectedKeyOutput, nil)
	mockSSM.On("GetParameter", mock.Anything, expectedSecretInput, mock.Anything).Return(expectedSecretOutput, expectedErr)

	krakenConfig := KrakenConf{}
	key, secret, err := krakenConfig.GetKrakenDetails(context.Background(), &mockSSM)

	mockSSM.AssertExpectations(t)
	assert.Nil(t, key)
	assert.Nil(t, secret)
	assert.NotNil(t, err)

	assert.Equal(t, expectedErr, err)
}

// Ensures when the key and secret can be retrieved
// they are returned
func TestGetKrakenDetails(t *testing.T) {
	expectedKeyInput := &ssm.GetParameterInput{Name: &SSMKrakenKey, WithDecryption: true}
	expectedSecretInput := &ssm.GetParameterInput{Name: &SSMKrakenSecret, WithDecryption: true}

	expectedKey := "key"
	expectedSecret := "secret"
	expectedKeyOutput := &ssm.GetParameterOutput{Parameter: &types.Parameter{Value: &expectedKey}}
	expectedSecretOutput := &ssm.GetParameterOutput{Parameter: &types.Parameter{Value: &expectedSecret}}

	mockSSM := pkg.MockSSMClient{}
	mockSSM.On("GetParameter", mock.Anything, expectedKeyInput, mock.Anything).Return(expectedKeyOutput, nil)
	mockSSM.On("GetParameter", mock.Anything, expectedSecretInput, mock.Anything).Return(expectedSecretOutput, nil)

	krakenConfig := KrakenConf{}
	key, secret, err := krakenConfig.GetKrakenDetails(context.Background(), &mockSSM)

	mockSSM.AssertExpectations(t)
	assert.Equal(t, expectedKey, *key)
	assert.Equal(t, expectedSecret, *secret)
	assert.Nil(t, err)
}
