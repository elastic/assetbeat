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
const resourceGroup2 = "WRONGVM"
const subscriptionId = "12cabcb4-86e8-404f-111111111111"
const instance1Name = "instance1"

var instanceid1 = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s", subscriptionId, resourceGroup1, instance1Name)

const instance2Name = "instance2"

var instanceid2 = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s", subscriptionId, resourceGroup1, instance2Name)

const instance3Name = "instance3"

var instanceid3 = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s", subscriptionId, resourceGroup1, instance3Name)

const instanceDiffResourceGroupName = "instanceDiffResourceGroup"

var instanceIdDiffResourceGroup = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s", subscriptionId, resourceGroup2, instanceDiffResourceGroupName)

var instance1 = armcompute.VirtualMachine{
	Location: to.Ptr("westeurope"),
	ID:       to.Ptr(instanceid1),
	Name:     to.Ptr(instance1Name),
}

var instance2 = armcompute.VirtualMachine{
	Location: to.Ptr("northeurope"),
	ID:       to.Ptr(instanceid2),
	Name:     to.Ptr(instance2Name),
}

var instance3 = armcompute.VirtualMachine{
	Location: to.Ptr("eastus"),
	ID:       to.Ptr(instanceid3),
	Name:     to.Ptr(instance3Name),
}

var instanceDiffResourceGroup = armcompute.VirtualMachine{
	Location: to.Ptr("northeurope"),
	ID:       to.Ptr(instanceIdDiffResourceGroup),
	Name:     to.Ptr(instanceDiffResourceGroupName),
}

func TestAssetsAzure_collectAzureAssets(t *testing.T) {
	for _, tt := range []struct {
		name           string
		regions        []string
		fakeServer     fake.VirtualMachinesServer
		subscriptionId string
		resourceGroup  string
		expectedEvents []beat.Event
	}{
		{
			name:           "Test with no regions specified and no resource group specified",
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
						"asset.ean":                     "host:" + instanceid1,
						"asset.id":                      instanceid1,
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
						"asset.ean":                     "host:" + instanceid2,
						"asset.id":                      instanceid2,
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
						"asset.ean":                     "host:" + instanceid3,
						"asset.id":                      instanceid3,
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
			name:           "Test with multiple regions specified but no resource group specified",
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
						"asset.ean":                     "host:" + instanceid1,
						"asset.id":                      instanceid1,
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
						"asset.ean":                     "host:" + instanceid2,
						"asset.id":                      instanceid2,
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
		{
			name:           "Test with multiple regions specified and resource group specified",
			regions:        []string{"westeurope", "northeurope"},
			resourceGroup:  resourceGroup1,
			subscriptionId: "12cabcb4-86e8-404f-111111111111",
			fakeServer: fake.VirtualMachinesServer{
				NewListAllPager: func(options *armcompute.VirtualMachinesClientListAllOptions) (resp azfake.PagerResponder[armcompute.VirtualMachinesClientListAllResponse]) {

					page := armcompute.VirtualMachinesClientListAllResponse{
						VirtualMachineListResult: armcompute.VirtualMachineListResult{
							Value: []*armcompute.VirtualMachine{
								&instance1,
								&instance2,
								&instance3,
								&instanceDiffResourceGroup,
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
						"asset.ean":                     "host:" + instanceid1,
						"asset.id":                      instanceid1,
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
						"asset.ean":                     "host:" + instanceid2,
						"asset.id":                      instanceid2,
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

			err = collectAzureVMAssets(ctx, client, tt.subscriptionId, tt.regions, tt.resourceGroup, logger, publisher)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedEvents, publisher.Events)
		})

	}
}
