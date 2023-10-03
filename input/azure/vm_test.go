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

package azure

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5/fake"
	"github.com/elastic/assetbeat/input/internal"
	"github.com/elastic/assetbeat/input/testutil"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

const resourceGroup1 = "TESTVM"
const subscriptionId = "12cabcb4-86e8-404f-111111111111"
const instance1Name = "instance1"

const instanceVMId1 = "1"

var instanceid1 = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s", subscriptionId, resourceGroup1, instance1Name)

const instance2Name = "instance2"
const instanceVMId2 = "2"

var instanceid2 = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s", subscriptionId, resourceGroup1, instance2Name)

const instance3Name = "instance3"
const instanceVMId3 = "3"

var instanceid3 = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s", subscriptionId, resourceGroup1, instance3Name)

const ss1Name = "vmss1"
const ssVm1Name = "vmss_0"
const ssVm2Name = "vmss_1"

var ssID = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachineScaleSets/%s", subscriptionId, resourceGroup1, ss1Name)

var instance1 = armcompute.VirtualMachine{
	Location:   to.Ptr("westeurope"),
	ID:         to.Ptr(instanceid1),
	Name:       to.Ptr(instance1Name),
	Properties: &armcompute.VirtualMachineProperties{VMID: to.Ptr(instanceVMId1)},
}

var instance2 = armcompute.VirtualMachine{
	Location:   to.Ptr("northeurope"),
	ID:         to.Ptr(instanceid2),
	Name:       to.Ptr(instance2Name),
	Properties: &armcompute.VirtualMachineProperties{VMID: to.Ptr(instanceVMId2)},
}

var instance3 = armcompute.VirtualMachine{
	Location:   to.Ptr("eastus"),
	ID:         to.Ptr(instanceid3),
	Name:       to.Ptr(instance3Name),
	Properties: &armcompute.VirtualMachineProperties{VMID: to.Ptr(instanceVMId3)},
}

var scaleSet = armcompute.VirtualMachineScaleSet{
	Location: to.Ptr("westeurope"),
	ID:       to.Ptr(ssID),
	Name:     to.Ptr(ss1Name),
}

var scaleSetVm1 = armcompute.VirtualMachineScaleSetVM{
	Location:   to.Ptr("westeurope"),
	InstanceID: to.Ptr("0"),
	Name:       to.Ptr(ssVm1Name),
	Properties: &armcompute.VirtualMachineScaleSetVMProperties{VMID: to.Ptr(instanceVMId1)},
}

var scaleSetVm2 = armcompute.VirtualMachineScaleSetVM{
	Location:   to.Ptr("northeurope"),
	InstanceID: to.Ptr("1"),
	Name:       to.Ptr(ssVm2Name),
	Properties: &armcompute.VirtualMachineScaleSetVMProperties{VMID: to.Ptr(instanceVMId2)},
}

var status1 = armcompute.InstanceViewStatus{
	DisplayStatus: to.Ptr("Provisioning"),
}
var status2 = armcompute.InstanceViewStatus{
	DisplayStatus: to.Ptr("VM Running"),
}

func TestAssetsAzure_collectAzureVMAssets(t *testing.T) {
	for _, tt := range []struct {
		name           string
		regions        []string
		fakeServer     fake.VirtualMachinesServer
		subscriptionId string
		expectedEvents []beat.Event
	}{
		{
			name:           "Test with no regions specified",
			subscriptionId: "12cabcb4-86e8-404f-111111111111",
			fakeServer: fake.VirtualMachinesServer{
				NewListAllPager: func(options *armcompute.VirtualMachinesClientListAllOptions) (resp azfake.PagerResponder[armcompute.VirtualMachinesClientListAllResponse]) {

					page := armcompute.VirtualMachinesClientListAllResponse{
						VirtualMachineListResult: armcompute.VirtualMachineListResult{
							Value: []*armcompute.VirtualMachine{
								&instance1,
								&instance2,
								&instance3,
							},
						},
					}
					resp.AddPage(http.StatusOK, page, nil)
					return
				},
			},
			expectedEvents: []beat.Event{
				{
					Fields: mapstr.M{
						"asset.ean":                     "host:" + instanceVMId1,
						"asset.id":                      instanceVMId1,
						"asset.name":                    instance1Name,
						"asset.type":                    "azure.vm.instance",
						"asset.kind":                    "host",
						"asset.metadata.state":          "",
						"asset.metadata.resource_group": "TESTVM",
						"cloud.account.id":              "12cabcb4-86e8-404f-111111111111",
						"cloud.provider":                "azure",
						"cloud.region":                  "westeurope",
					},
					Meta: mapstr.M{
						"index": internal.GetDefaultIndexName(),
					},
				},
				{
					Fields: mapstr.M{
						"asset.ean":                     "host:" + instanceVMId2,
						"asset.id":                      instanceVMId2,
						"asset.name":                    instance2Name,
						"asset.type":                    "azure.vm.instance",
						"asset.kind":                    "host",
						"asset.metadata.state":          "",
						"asset.metadata.resource_group": "TESTVM",
						"cloud.account.id":              "12cabcb4-86e8-404f-111111111111",
						"cloud.provider":                "azure",
						"cloud.region":                  "northeurope",
					},
					Meta: mapstr.M{
						"index": internal.GetDefaultIndexName(),
					},
				},
				{
					Fields: mapstr.M{
						"asset.ean":                     "host:" + instanceVMId3,
						"asset.id":                      instanceVMId3,
						"asset.name":                    instance3Name,
						"asset.type":                    "azure.vm.instance",
						"asset.kind":                    "host",
						"asset.metadata.state":          "",
						"asset.metadata.resource_group": "TESTVM",
						"cloud.account.id":              "12cabcb4-86e8-404f-111111111111",
						"cloud.provider":                "azure",
						"cloud.region":                  "eastus",
					},
					Meta: mapstr.M{
						"index": internal.GetDefaultIndexName(),
					},
				},
			},
		},
		{
			name:           "Test with multiple regions specified",
			regions:        []string{"westeurope", "northeurope"},
			subscriptionId: "12cabcb4-86e8-404f-111111111111",
			fakeServer: fake.VirtualMachinesServer{
				NewListAllPager: func(options *armcompute.VirtualMachinesClientListAllOptions) (resp azfake.PagerResponder[armcompute.VirtualMachinesClientListAllResponse]) {

					page := armcompute.VirtualMachinesClientListAllResponse{
						VirtualMachineListResult: armcompute.VirtualMachineListResult{
							Value: []*armcompute.VirtualMachine{
								&instance1,
								&instance2,
								&instance3,
							},
						},
					}
					resp.AddPage(http.StatusOK, page, nil)
					return
				},
			},
			expectedEvents: []beat.Event{
				{
					Fields: mapstr.M{
						"asset.ean":                     "host:" + instanceVMId1,
						"asset.id":                      instanceVMId1,
						"asset.name":                    instance1Name,
						"asset.type":                    "azure.vm.instance",
						"asset.kind":                    "host",
						"asset.metadata.state":          "",
						"asset.metadata.resource_group": "TESTVM",
						"cloud.account.id":              "12cabcb4-86e8-404f-111111111111",
						"cloud.provider":                "azure",
						"cloud.region":                  "westeurope",
					},
					Meta: mapstr.M{
						"index": internal.GetDefaultIndexName(),
					},
				},
				{
					Fields: mapstr.M{
						"asset.ean":                     "host:" + instanceVMId2,
						"asset.id":                      instanceVMId2,
						"asset.name":                    instance2Name,
						"asset.type":                    "azure.vm.instance",
						"asset.kind":                    "host",
						"asset.metadata.state":          "",
						"asset.metadata.resource_group": "TESTVM",
						"cloud.account.id":              "12cabcb4-86e8-404f-111111111111",
						"cloud.provider":                "azure",
						"cloud.region":                  "northeurope",
					},
					Meta: mapstr.M{
						"index": internal.GetDefaultIndexName(),
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			publisher := testutil.NewInMemoryPublisher()

			ctx := context.Background()
			logger := logp.NewLogger("test")

			client, err := armcompute.NewVirtualMachinesClient("subscriptionID", azfake.NewTokenCredential(), &arm.ClientOptions{
				ClientOptions: azcore.ClientOptions{
					Transport: fake.NewVirtualMachinesServerTransport(&tt.fakeServer),
				},
			})
			assert.NoError(t, err)

			err = collectAzureVMAssets(ctx, client, tt.subscriptionId, tt.regions, logger, publisher)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedEvents, publisher.Events)
		})

	}
}

func TestAssetsAzure_collectAzureScaleSetsVMAssets(t *testing.T) {
	for _, tt := range []struct {
		name           string
		regions        []string
		fakeSSServer   fake.VirtualMachineScaleSetsServer
		fakeVMServer   fake.VirtualMachineScaleSetVMsServer
		subscriptionId string
		expectedEvents []beat.Event
	}{
		{
			name:           "Test with one ScaleSet with tow instances",
			subscriptionId: "12cabcb4-86e8-404f-111111111111",
			fakeSSServer: fake.VirtualMachineScaleSetsServer{
				NewListAllPager: func(options *armcompute.VirtualMachineScaleSetsClientListAllOptions) (resp azfake.PagerResponder[armcompute.VirtualMachineScaleSetsClientListAllResponse]) {

					page := armcompute.VirtualMachineScaleSetsClientListAllResponse{
						VirtualMachineScaleSetListWithLinkResult: armcompute.VirtualMachineScaleSetListWithLinkResult{
							Value: []*armcompute.VirtualMachineScaleSet{
								&scaleSet,
							},
						},
					}
					resp.AddPage(http.StatusOK, page, nil)
					return
				},
			},
			fakeVMServer: fake.VirtualMachineScaleSetVMsServer{
				NewListPager: func(resourceGroup string, vmScaleSetName string, options *armcompute.VirtualMachineScaleSetVMsClientListOptions) (resp azfake.PagerResponder[armcompute.VirtualMachineScaleSetVMsClientListResponse]) {

					page := armcompute.VirtualMachineScaleSetVMsClientListResponse{
						VirtualMachineScaleSetVMListResult: armcompute.VirtualMachineScaleSetVMListResult{
							Value: []*armcompute.VirtualMachineScaleSetVM{
								&scaleSetVm1,
								&scaleSetVm2,
							},
						},
					}
					resp.AddPage(http.StatusOK, page, nil)
					return
				},
				GetInstanceView: func(ctx context.Context, resourceGroupName, vmScaleSetName, instanceId string, options *armcompute.VirtualMachineScaleSetVMsClientGetInstanceViewOptions) (resp azfake.Responder[armcompute.VirtualMachineScaleSetVMsClientGetInstanceViewResponse], errResp azfake.ErrorResponder) {

					response := armcompute.VirtualMachineScaleSetVMsClientGetInstanceViewResponse{
						VirtualMachineScaleSetVMInstanceView: armcompute.VirtualMachineScaleSetVMInstanceView{
							Statuses: []*armcompute.InstanceViewStatus{
								&status1,
								&status2,
							},
						},
					}
					resp.SetResponse(http.StatusOK, response, nil)
					return
				},
			},
			expectedEvents: []beat.Event{
				{
					Fields: mapstr.M{
						"asset.ean":                     "host:" + instanceVMId1,
						"asset.id":                      instanceVMId1,
						"asset.name":                    ssVm1Name,
						"asset.type":                    "azure.vm.instance",
						"asset.kind":                    "host",
						"asset.metadata.state":          "VM Running",
						"asset.metadata.resource_group": "TESTVM",
						"cloud.account.id":              "12cabcb4-86e8-404f-111111111111",
						"cloud.provider":                "azure",
						"cloud.region":                  "westeurope",
					},
					Meta: mapstr.M{
						"index": internal.GetDefaultIndexName(),
					},
				},
				{
					Fields: mapstr.M{
						"asset.ean":                     "host:" + instanceVMId2,
						"asset.id":                      instanceVMId2,
						"asset.name":                    ssVm2Name,
						"asset.type":                    "azure.vm.instance",
						"asset.kind":                    "host",
						"asset.metadata.state":          "VM Running",
						"asset.metadata.resource_group": "TESTVM",
						"cloud.account.id":              "12cabcb4-86e8-404f-111111111111",
						"cloud.provider":                "azure",
						"cloud.region":                  "northeurope",
					},
					Meta: mapstr.M{
						"index": internal.GetDefaultIndexName(),
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			publisher := testutil.NewInMemoryPublisher()

			ctx := context.Background()
			logger := logp.NewLogger("test")

			vmclient, _ := armcompute.NewVirtualMachineScaleSetVMsClient("subscriptionID", azfake.NewTokenCredential(), &arm.ClientOptions{
				ClientOptions: azcore.ClientOptions{
					Transport: fake.NewVirtualMachineScaleSetVMsServerTransport(&tt.fakeVMServer),
				},
			})

			ssclient, err := armcompute.NewVirtualMachineScaleSetsClient("subscriptionID", azfake.NewTokenCredential(), &arm.ClientOptions{
				ClientOptions: azcore.ClientOptions{
					Transport: fake.NewVirtualMachineScaleSetsServerTransport(&tt.fakeSSServer),
				},
			})
			assert.NoError(t, err)

			err = collectAzureScaleSetsVMAssets(ctx, vmclient, ssclient, tt.subscriptionId, tt.regions, logger, publisher)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedEvents, publisher.Events)
		})

	}
}
