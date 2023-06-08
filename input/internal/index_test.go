package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/assetbeat/input/testutil"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestIndex_WithIndex(t *testing.T) {
	for _, tt := range []struct {
		name     string
		assetOp  AssetOption
		expected beat.Event
	}{
		{
			name:    "no defined index namespace, the default is used",
			assetOp: WithIndex("aws.ec2.instance", ""),
			expected: beat.Event{
				Fields: mapstr.M{},
				Meta:   mapstr.M{"index": "assets-aws.ec2.instance-default"},
			},
		},
		{
			name:    "defined namespace",
			assetOp: WithIndex("aws.ec2.instance", "test.namespace"),
			expected: beat.Event{
				Fields: mapstr.M{},
				Meta:   mapstr.M{"index": "assets-aws.ec2.instance-test.namespace"},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			publisher := testutil.NewInMemoryPublisher()

			Publish(publisher, tt.assetOp)

			assert.Equal(t, 1, len(publisher.Events))
			assert.Equal(t, tt.expected, publisher.Events[0])
		})
	}
}
