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
	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"context"
	"encoding/json"
	"github.com/gogo/protobuf/proto"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/option"
)

var findComputeProjectRe = regexp.MustCompile("/projects/([a-z_]+)/aggregated/instances")

func TestGetAllComputeInstances(t *testing.T) {
	for _, tt := range []struct {
		name string

		ctx           context.Context
		cfg           config
		httpResponses map[string]computepb.InstanceAggregatedList

		expectedInstances []computeInstance
	}{
		{
			name: "with no project specified",

			ctx: context.Background(),
			cfg: config{},
		},
		{
			name: "with one project specified",

			ctx: context.Background(),
			cfg: config{
				Projects: []string{"my_project"},
			},

			httpResponses: map[string]computepb.InstanceAggregatedList{
				"my_project": {
					Items: map[string]*computepb.InstancesScopedList{
						"europe-central2-a": {
							Instances: []*computepb.Instance{
								{
									Id:   proto.Uint64(1),
									Zone: proto.String("https://www.googleapis.com/compute/v1/projects/my_project/zones/europe-west1-d"),
									NetworkInterfaces: []*computepb.NetworkInterface{
										&computepb.NetworkInterface{
											Network: proto.String("https://www.googleapis.com/compute/v1/projects/my_project/global/networks/my_network"),
										},
									},
									Status: proto.String("RUNNING"),
								},
							},
						}},
				},
			},

			expectedInstances: []computeInstance{
				computeInstance{
					ID:      "1",
					Region:  "europe-west1",
					Account: "my_project",
					VPCs:    []string{"my_network"},
					Metadata: mapstr.M{
						"state": "RUNNING",
					},
				},
			},
		},
		//{
		//	name: "with multiple projects specified",
		//
		//	ctx: context.Background(),
		//	cfg: config{
		//		Projects: []string{"my_project", "my_second_project"},
		//	},
		//
		//	httpResponses: map[string]computepb.InstanceAggregatedList{
		//		"my_project": computepb.InstanceAggregatedList{
		//			Items: map[string]computepb.InstancesScopedList{
		//				"europe-central2-a": computepb.InstancesScopedList{
		//					Instances: []*computepb.Instance{
		//						&computepb.Instance{
		//							Id:     1,
		//							Zone:   "https://www.googleapis.com/compute/v1/projects/my_project/zones/europe-west1-d",
		//							Status: "PROVISIONING",
		//						},
		//					},
		//				},
		//			},
		//		},
		//		"my_second_project": computepb.InstanceAggregatedList{
		//			Items: map[string]computepb.InstancesScopedList{
		//				"europe-central2-a": computepb.InstancesScopedList{
		//					Instances: []*computepb.Instance{
		//						&computepb.Instance{
		//							Id:     42,
		//							Zone:   "https://www.googleapis.com/compute/v1/projects/my_project/zones/europe-west1-d",
		//							Status: "STOPPED",
		//						},
		//					},
		//				},
		//			},
		//		},
		//	},
		//
		//	expectedInstances: []computeInstance{
		//		computeInstance{
		//			ID:      "1",
		//			Region:  "europe-west1",
		//			Account: "my_project",
		//			Metadata: mapstr.M{
		//				"state": "PROVISIONING",
		//			},
		//		},
		//		computeInstance{
		//			ID:      "42",
		//			Region:  "europe-west1",
		//			Account: "my_second_project",
		//			Metadata: mapstr.M{
		//				"state": "STOPPED",
		//			},
		//		},
		//	},
		//},
		//{
		//	name: "with a region filter",
		//
		//	ctx: context.Background(),
		//	cfg: config{
		//		Projects: []string{"my_project"},
		//		Regions:  []string{"us-west1"},
		//	},
		//
		//	httpResponses: map[string]computepb.InstanceAggregatedList{
		//		"my_project": computepb.InstanceAggregatedList{
		//			Items: map[string]computepb.InstancesScopedList{
		//				"europe-central2-a": computepb.InstancesScopedList{
		//					Instances: []*computepb.Instance{
		//						&computepb.Instance{
		//							Id:   1,
		//							Zone: "https://www.googleapis.com/compute/v1/projects/my_project/zones/europe-west1-d",
		//							NetworkInterfaces: []*computepb.NetworkInterface{
		//								&computepb.NetworkInterface{
		//									Network: "https://www.googleapis.com/compute/v1/projects/my_project/global/networks/my_network",
		//								},
		//							},
		//							Status: "RUNNING",
		//						},
		//					},
		//				},
		//				"us-west1-b": computepb.InstancesScopedList{
		//					Instances: []*computepb.Instance{
		//						&computepb.Instance{
		//							Id:   2,
		//							Zone: "https://www.googleapis.com/compute/v1/projects/my_project/zones/us-west1-b",
		//							NetworkInterfaces: []*computepb.NetworkInterface{
		//								&computepb.NetworkInterface{
		//									Network: "https://www.googleapis.com/compute/v1/projects/my_project/global/networks/my_network",
		//								},
		//							},
		//							Status: "RUNNING",
		//						},
		//					},
		//				},
		//			},
		//		},
		//	},
		//
		//	expectedInstances: []computeInstance{
		//		computeInstance{
		//			ID:      "2",
		//			Region:  "us-west1",
		//			Account: "my_project",
		//			VPCs:    []string{"my_network"},
		//			Metadata: mapstr.M{
		//				"state": "RUNNING",
		//			},
		//		},
		//	},
		//},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				m := findComputeProjectRe.FindStringSubmatch(r.URL.Path)
				if len(m) < 2 {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				project := m[1]

				b, err := json.Marshal(tt.httpResponses[project])
				assert.NoError(t, err)
				_, err = w.Write(b)
				assert.NoError(t, err)
			}))
			defer ts.Close()

			svc, err := compute.NewInstancesRESTClient(tt.ctx, option.WithoutAuthentication(), option.WithEndpoint(ts.URL))
			assert.NoError(t, err)

			instances, err := getAllComputeInstances(tt.ctx, tt.cfg, svc)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedInstances, instances)
		})
	}
}
