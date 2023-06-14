package hostdata

import (
	"context"
	"github.com/elastic/assetbeat/input/testutil"
	"github.com/elastic/elastic-agent-libs/logp"
	"testing"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/stretchr/testify/assert"
)

func TestHostdata_configurationAndInitialization(t *testing.T) {
	input, err := configure(conf.NewConfig())
	assert.Nil(t, err)

	hostdata := input.(*hostdata)
	assert.Equal(t, defaultCollectionPeriod, hostdata.config.Period)

	assert.NotEmpty(t, hostdata.hostInfo)
	hostID, _ := hostdata.hostInfo.GetValue("host.id")
	assert.NotEmpty(t, hostID)
}

func TestHostdata_reportHostDataAssets(t *testing.T) {
	input, _ := configure(conf.NewConfig())

	publisher := testutil.NewInMemoryPublisher()
	input.(*hostdata).reportHostDataAssets(context.Background(), logp.NewLogger("test"), publisher)
	assert.NotEmpty(t, publisher.Events)
	event := publisher.Events[0]

	// check that the base fields are populated
	hostID, _ := event.Fields.GetValue("host.id")
	assetID, _ := event.Fields.GetValue("asset.id")
	assetType, _ := event.Fields.GetValue("asset.type")
	assetKind, _ := event.Fields.GetValue("asset.kind")
	destinationDatastream, _ := event.Meta.GetValue("index")

	assert.NotEmpty(t, hostID)
	assert.Equal(t, hostID, assetID)
	assert.Equal(t, "host", assetType)
	assert.Equal(t, "host", assetKind)
	assert.Equal(t, "assets-host-default", destinationDatastream)

	// check that the networking fields are populated
	// (and that the stored host data has not been modified)
	ips, _ := event.Fields.GetValue("host.ip")
	assert.NotEmpty(t, ips)

	_, err := input.(*hostdata).hostInfo.GetValue("host.ip")
	assert.Error(t, err)
}
