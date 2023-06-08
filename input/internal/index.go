package internal

import (
	"fmt"
	"github.com/elastic/beats/v7/libbeat/beat"
)

// Assets data is published to indexes following the same name pattern used in Agent
// type-dataset-namespace, and has its own index type.
const indexType = "assets"
const indexDefaultNamespace = "default"

func WithIndex(assetType string, indexNamespace string) AssetOption {
	ns := indexDefaultNamespace
	if indexNamespace != "" {
		ns = indexNamespace
	}
	return func(e beat.Event) beat.Event {
		e.Meta["index"] = fmt.Sprintf("%s-%s-%s", indexType, assetType, ns)
		return e
	}
}
