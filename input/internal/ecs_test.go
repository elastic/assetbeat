package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/assetbeat/input/testutil"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestECS_WithCloudInstanceId(t *testing.T) {
	for _, tt := range []struct {
		name     string
		assetOp  AssetOption
		expected beat.Event
	}{
		{
			name:    "instance Id provided",
			assetOp: WithCloudInstanceId("i-0699b78f46f0fa248"),
			expected: beat.Event{
				Fields: mapstr.M{"cloud.instance.id": "i-0699b78f46f0fa248"},
				Meta:   mapstr.M{},
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
