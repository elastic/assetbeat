package assets_aws

import (
	"context"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/inputrunner/mocks"
	"github.com/golang/mock/gomock"
)

func TestAssetsAWS_publishAWSAsset_IncludesRequiredFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	publisher := mocks.NewMockPublisher(ctrl)
	expectedAsset := mapstr.M{
		"cloud.provider":   "aws",
		"cloud.region":     "eu-west-1",
		"cloud.account.id": "1234",
		"asset.type":       "aws.ec2.instance",
		"asset.id":         "i-1234",
		"asset.ean":        "aws.ec2.instance:i-1234",
	}
	publisher.EXPECT().Publish(beat.Event{Fields: expectedAsset})
	publishAWSAsset(publisher, "eu-west-1", "1234", "aws.ec2.instance", "i-1234", nil, nil, nil)
}

func TestAssetAWS_getConfigForRegion_GivenExplicitCredsInConfig_CreatesCorrectAWSConfig(t *testing.T) {
	ctx := context.Background()
	inputCfg := Config{
		Regions:         []string{"eu-west-2", "eu-west-1"},
		AccessKeyId:     "accesskey123",
		SecretAccessKey: "secretkey123",
		SessionToken:    "token123",
		Period:          time.Second * 600,
	}
	region := "eu-west-2"
	awsCfg, err := getAWSConfigForRegion(ctx, inputCfg, region)
	assert.NoError(t, err)
	retrievedAWSCreds, err := awsCfg.Credentials.Retrieve(context.Background())
	assert.NoError(t, err)

	assert.Equal(t, inputCfg.AccessKeyId, retrievedAWSCreds.AccessKeyID)
	assert.Equal(t, inputCfg.SecretAccessKey, retrievedAWSCreds.SecretAccessKey)
	assert.Equal(t, inputCfg.SessionToken, retrievedAWSCreds.SessionToken)
	assert.Equal(t, region, awsCfg.Region)
}

func TestAssetAWS_getConfigForRegion_GivenLocalCreds_CreatesCorrectAWSConfig(t *testing.T) {
	ctx := context.Background()
	accessKey := "EXAMPLE_ACCESS_KEY"
	secretKey := "EXAMPLE_SECRETE_KEY"
	os.Setenv("AWS_ACCESS_KEY", accessKey)
	os.Setenv("AWS_SECRET_ACCESS_KEY", secretKey)
	inputCfg := Config{
		Regions:         []string{"eu-west-2", "eu-west-1"},
		AccessKeyId:     "",
		SecretAccessKey: "",
		SessionToken:    "",
		Period:          time.Second * 600,
	}
	region := "eu-west-2"
	awsCfg, err := getAWSConfigForRegion(ctx, inputCfg, region)
	assert.NoError(t, err)
	retrievedAWSCreds, err := awsCfg.Credentials.Retrieve(context.Background())
	assert.NoError(t, err)

	assert.Equal(t, accessKey, retrievedAWSCreds.AccessKeyID)
	assert.Equal(t, secretKey, retrievedAWSCreds.SecretAccessKey)
	assert.Equal(t, region, awsCfg.Region)
}
