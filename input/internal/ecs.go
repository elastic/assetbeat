package internal

import (
	"github.com/elastic/beats/v7/libbeat/beat"
)

func WithCloudInstanceId(instanceId string) AssetOption {
	return func(e beat.Event) beat.Event {
		e.Fields["cloud.instance.id"] = instanceId
		return e
	}
}
