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
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/elastic/assetbeat/input/internal"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"strings"
)

type AzureVMInstance struct {
	ID             string
	Name           string
	SubscriptionID string
	Region         string
	Tags           map[string]*string
	Metadata       mapstr.M
}

func collectAzureVMAssets(ctx context.Context, client *armcompute.VirtualMachinesClient, subscriptionId string, regions []string, resourceGroup string, log *logp.Logger, publisher stateless.Publisher) error {

	instances, err := getAllAzureVMInstances(ctx, client, subscriptionId, regions, resourceGroup)
	if err != nil {
		return err
	}

	assetType := "azure.vm.instance"
	assetKind := "host"
	log.Debug("Publishing Azure VM instances")

	for _, instance := range instances {
		options := []internal.AssetOption{
			internal.WithAssetCloudProvider("azure"),
			internal.WithAssetRegion(instance.Region),
			internal.WithAssetAccountID(instance.SubscriptionID),
			internal.WithAssetKindAndID(assetKind, instance.ID),
			internal.WithAssetType(assetType),
			internal.WithAssetMetadata(instance.Metadata),
		}
		if instance.Name != "" {
			options = append(options, internal.WithAssetName(instance.Name))
		}
		internal.Publish(publisher, nil, options...)
	}

	return nil
}

func getAllAzureVMInstances(ctx context.Context, client *armcompute.VirtualMachinesClient, subscriptionId string, regions []string, resourceGroup string) ([]AzureVMInstance, error) {
	var vmInstances []AzureVMInstance
	pager := client.NewListAllPager(&armcompute.VirtualMachinesClientListAllOptions{StatusOnly: to.Ptr("true")})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to advance page: %v", err)
		}
		for _, v := range page.Value {
			if wantRegion(v, regions) && wantResourceGroup(v, resourceGroup) {
				var status string
				if v.Properties != nil && v.Properties.InstanceView != nil && len(v.Properties.InstanceView.Statuses) > 1 {
					status = *v.Properties.InstanceView.Statuses[1].DisplayStatus
				}
				vmInstance := AzureVMInstance{
					ID:             *v.Properties.VMID,
					Name:           *v.Name,
					SubscriptionID: subscriptionId,
					Region:         *v.Location,
					Tags:           v.Tags,
					Metadata: mapstr.M{
						"state":          status,
						"resource_group": getResourceGroupFromId(*v.ID),
					},
				}
				vmInstances = append(vmInstances, vmInstance)
			}
		}
	}
	return vmInstances, nil
}

func wantResourceGroup(v *armcompute.VirtualMachine, resourceGroup string) bool {
	if resourceGroup == "" {
		return true
	}
	if getResourceGroupFromId(*v.ID) == resourceGroup {
		return true
	}
	return false
}

func wantRegion(v *armcompute.VirtualMachine, regions []string) bool {
	if len(regions) == 0 {
		return true
	}
	for _, region := range regions {
		if *v.Location == region {
			return true
		}
	}
	return false
}

func getResourceGroupFromId(res string) string {
	s := strings.Split(res, "/")
	return s[4]
}
