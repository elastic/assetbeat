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

package gcp

import (
	"context"
	"strconv"

	"github.com/elastic/elastic-agent-libs/mapstr"
	stateless "github.com/elastic/inputrunner/input/v2/input-stateless"
	"google.golang.org/api/compute/v1"
)

type computeInstance struct {
	ID       string
	Region   string
	Metadata mapstr.M
}

func collectComputeAssets(ctx context.Context, cfg config, publisher stateless.Publisher) error {
	svc, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	instances, err := getAllComputeInstances(ctx, cfg, svc)
	if err != nil {
		return err
	}

	for _, instance := range instances {
		publishAsset(
			publisher,
			instance.Region,
			"gcp.compute.instance",
			instance.ID,
			nil,
			nil,
			instance.Metadata,
		)
	}

	return nil
}

func getAllComputeInstances(ctx context.Context, cfg config, svc *compute.Service) ([]computeInstance, error) {
	var instances []computeInstance

	for _, p := range cfg.Config.Projects {
		req := svc.Instances.AggregatedList(p)

		err := req.Pages(ctx, func(page *compute.InstanceAggregatedList) error {
			for _, isl := range page.Items {
				for _, i := range isl.Instances {
					instances = append(instances, computeInstance{
						ID:     strconv.FormatUint(i.Id, 10),
						Region: i.Zone,
						Metadata: mapstr.M{
							"state": string(i.Status),
						},
					})
				}
			}
			return nil
		})

		if err != nil {
			return instances, err
		}
	}

	return instances, nil
}
