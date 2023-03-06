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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

var findProjectRe = regexp.MustCompile("/projects/([a-z_]+)/aggregated/instances")

func TestGetAllComputeInstances(t *testing.T) {
	for _, tt := range []struct {
		name string

		ctx           context.Context
		cfg           config
		httpResponses map[string]compute.InstanceAggregatedList

		expectedInstances []computeInstance
	}{
		{
			name: "with no project specified",

			ctx: context.Background(),
			cfg: config{
				Config{},
			},
		},
		{
			name: "with one project specified",

			ctx: context.Background(),
			cfg: config{
				Config{
					Projects: []string{"my_project"},
				},
			},

			httpResponses: map[string]compute.InstanceAggregatedList{
				"my_project": compute.InstanceAggregatedList{
					Items: map[string]compute.InstancesScopedList{
						"europe-central2-a": compute.InstancesScopedList{
							Instances: []*compute.Instance{
								&compute.Instance{
									Id:     1,
									Status: "RUNNING",
								},
							},
						},
					},
				},
			},

			expectedInstances: []computeInstance{
				computeInstance{
					ID: "1",
					Metadata: mapstr.M{
						"state": "RUNNING",
					},
				},
			},
		},
		{
			name: "with multiple projects specified",

			ctx: context.Background(),
			cfg: config{
				Config{
					Projects: []string{"my_project", "my_second_project"},
				},
			},

			httpResponses: map[string]compute.InstanceAggregatedList{
				"my_project": compute.InstanceAggregatedList{
					Items: map[string]compute.InstancesScopedList{
						"europe-central2-a": compute.InstancesScopedList{
							Instances: []*compute.Instance{
								&compute.Instance{
									Id:     1,
									Status: "PROVISIONING",
								},
							},
						},
					},
				},
				"my_second_project": compute.InstanceAggregatedList{
					Items: map[string]compute.InstancesScopedList{
						"europe-central2-a": compute.InstancesScopedList{
							Instances: []*compute.Instance{
								&compute.Instance{
									Id:     42,
									Status: "STOPPED",
								},
							},
						},
					},
				},
			},

			expectedInstances: []computeInstance{
				computeInstance{
					ID: "1",
					Metadata: mapstr.M{
						"state": "PROVISIONING",
					},
				},
				computeInstance{
					ID: "42",
					Metadata: mapstr.M{
						"state": "STOPPED",
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				m := findProjectRe.FindStringSubmatch(r.URL.Path)
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

			svc, err := compute.NewService(tt.ctx, option.WithoutAuthentication(), option.WithEndpoint(ts.URL))
			assert.NoError(t, err)

			instances, err := getAllComputeInstances(tt.ctx, tt.cfg, svc)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedInstances, instances)
		})
	}
}
