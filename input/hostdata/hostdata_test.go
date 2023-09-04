// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package hostdata

import (
	"context"
	"github.com/elastic/assetbeat/input/internal"
	"testing"

	"github.com/elastic/go-sysinfo"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/host"

	"github.com/elastic/assetbeat/input/testutil"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/stretchr/testify/assert"

	conf "github.com/elastic/elastic-agent-libs/config"
)

type fakeCloudMetadataProcessor struct{}

func (p fakeCloudMetadataProcessor) Run(event *beat.Event) (*beat.Event, error) {
	_, err := event.PutValue("cloud.instance.id", "i-12342")
	if err != nil {
		return nil, err
	}
	return event, err
}

func (p fakeCloudMetadataProcessor) String() string {
	return "fake_add_cloud_metadata"
}

func TestHostdata_configurationAndInitialization(t *testing.T) {
	input, err := configure(conf.NewConfig())
	assert.Nil(t, err)

	hostdata := input.(*hostdata)
	assert.Equal(t, defaultCollectionPeriod, hostdata.config.Period)

	assert.NotEmpty(t, hostdata.hostInfo)
	hostname, _ := hostdata.hostInfo.GetValue("host.hostname")
	assert.NotEmpty(t, hostname)
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

	//assert.NotEmpty(t, hostID)
	assert.Equal(t, hostID, assetID)
	assert.Equal(t, "host", assetType)
	assert.Equal(t, "host", assetKind)
	assert.Equal(t, internal.GetDefaultIndexName(), destinationDatastream)

	// check that the networking fields are populated
	// (and that the stored host data has not been modified)
	ips, _ := event.Fields.GetValue("host.ip")
	assert.NotEmpty(t, ips)

	_, err := input.(*hostdata).hostInfo.GetValue("host.ip")
	assert.Error(t, err)
}

func TestHostdata_reportHostDataAssetsWithCloudMeta(t *testing.T) {
	input, _ := configure(conf.NewConfig())
	hostDataProvider, _ := sysinfo.Host()

	hd := hostdata{
		config:                    defaultConfig(),
		hostInfo:                  host.MapHostInfo(hostDataProvider.Info()),
		addCloudMetadataProcessor: fakeCloudMetadataProcessor{},
	}
	publisher := testutil.NewInMemoryPublisher()
	hd.reportHostDataAssets(context.Background(), logp.NewLogger("test"), publisher)
	assert.NotEmpty(t, publisher.Events)
	event := publisher.Events[0]

	// check that the base fields are populated
	hostID, _ := event.Fields.GetValue("host.id")
	assetID, _ := event.Fields.GetValue("asset.id")
	assetType, _ := event.Fields.GetValue("asset.type")
	assetKind, _ := event.Fields.GetValue("asset.kind")
	destinationDatastream, _ := event.Meta.GetValue("index")
	cloudID, _ := event.Fields.GetValue("cloud.instance.id")

	assert.NotEmpty(t, hostID)
	assert.NotEmpty(t, cloudID)
	assert.Equal(t, cloudID, assetID)
	assert.Equal(t, hostID, assetID)
	assert.Equal(t, "host", assetType)
	assert.Equal(t, "host", assetKind)
	assert.Equal(t, internal.GetDefaultIndexName(), destinationDatastream)

	// check that the networking fields are populated
	// (and that the stored host data has not been modified)
	ips, _ := event.Fields.GetValue("host.ip")
	assert.NotEmpty(t, ips)

	_, err := input.(*hostdata).hostInfo.GetValue("host.ip")
	assert.Error(t, err)
}
