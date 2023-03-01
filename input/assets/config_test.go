package assets

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAssets_IsTypeEnabled(t *testing.T) {
	for _, tt := range []struct {
		name            string
		shouldBeEnabled bool
		configuredTypes []string
		currentType     string
	}{
		{
			name:            "always enabled when config field is nil",
			shouldBeEnabled: true,
			configuredTypes: nil,
			currentType:     "pod", // doesn't matter
		},
		{
			name:            "always enabled when config field is empty",
			shouldBeEnabled: true,
			configuredTypes: []string{},
			currentType:     "ec2", // doesn't matter
		},
		{
			name:            "enabled for listed type",
			shouldBeEnabled: true,
			configuredTypes: []string{"vpc"},
			currentType:     "vpc",
		},
		{
			name:            "disabled when type isn't in the list",
			shouldBeEnabled: false,
			configuredTypes: []string{"eks"},
			currentType:     "node",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.shouldBeEnabled, IsTypeEnabled(tt.configuredTypes, tt.currentType))
		})
	}
}
