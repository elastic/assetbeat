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
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/elastic/assetbeat/input/internal"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/elastic-agent-libs/logp"
)

type AzureVMInstance struct {
	ID             string
	Name           string
	SubscriptionID string
	Region         string
	Tags           map[string]*string
}

func collectAzureVMAssets(ctx context.Context, cfg config, cred *azidentity.DefaultAzureCredential, log *logp.Logger, publisher stateless.Publisher) error {
	clientFactory, err := armcompute.NewClientFactory(cfg.SubscriptionID, cred, nil)
	if err != nil {
		return err
	}
	client := clientFactory.NewVirtualMachinesClient()
	instances, err := getAllAzureVMInstances(ctx, client, cfg.SubscriptionID, log)
	if err != nil {
		return err
	}

	assetType := "azure.vm.instance"
	assetKind := "host"
	log.Debug("Publishing Azure VM instances")

	for _, instance := range instances {
		var parents []string
		internal.Publish(publisher, nil,
			internal.WithAssetCloudProvider("azure"),
			internal.WithAssetRegion(instance.Region),
			internal.WithAssetAccountID(instance.SubscriptionID),
			internal.WithAssetKindAndID(assetKind, instance.ID),
			internal.WithAssetType(assetType),
			internal.WithAssetParents(parents),
		)
	}

	return nil
}

func getAllAzureVMInstances(ctx context.Context, client *armcompute.VirtualMachinesClient, subscriptionId string, log *logp.Logger) ([]AzureVMInstance, error) {
	var vmInstances []AzureVMInstance
	pager := client.NewListAllPager(&armcompute.VirtualMachinesClientListAllOptions{StatusOnly: nil,
		Filter: nil,
		Expand: nil,
	})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to advance page: %v", err)
		}
		for _, v := range page.Value {
			vmInstance := AzureVMInstance{
				ID:             *v.ID,
				Name:           *v.Name,
				SubscriptionID: subscriptionId,
				Region:         *v.Location,
				Tags:           v.Tags,
			}
			vmInstances = append(vmInstances, vmInstance)
		}
	}
	return vmInstances, nil
}
