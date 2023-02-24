package assets_aws

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAssetsAWS_flattenEC2Tags(t *testing.T) {
	tag1, tag2, a, b := "tag1", "tag2", "a", "b"
	tags := []types.Tag{{Key: &tag1, Value: &a}, {Key: &tag2, Value: &b}}
	flat := flattenEC2Tags(tags)
	expected := map[string]string{"tag1": "a", "tag2": "b"}
	assert.Equal(t, expected, flat)
}
