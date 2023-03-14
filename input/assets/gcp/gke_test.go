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
	"google.golang.org/api/container/v1"
	"google.golang.org/api/option"
)

var findGKEProjectRe = regexp.MustCompile("/projects/([a-z_-]+)/zones/([0-9a-z_,-]+)/clusters")

func TestGetAllGKEClusters(t *testing.T) {
	for _, tt := range []struct {
		name string

		ctx           context.Context
		cfg           config
		httpResponses map[string]container.ListClustersResponse

		expectedClusters []containerCluster
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

			httpResponses: map[string]container.ListClustersResponse{
				"my_project": container.ListClustersResponse{
					Clusters: []*container.Cluster{
						&container.Cluster{
							Id:      "1",
							Zone:    "https://www.googleapis.com/compute/v1/projects/my_project/zones/europe-west1-d",
							Network: "my_network",
							Status:  "RUNNING",
						},
					},
				},
			},

			expectedClusters: []containerCluster{
				containerCluster{
					ID:      "1",
					Region:  "europe-west1",
					Account: "my_project",
					VPC:     "my_network",
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
				Projects: []string{"my_project", "my_second_project"},
			},

			httpResponses: map[string]container.ListClustersResponse{
				"my_project": container.ListClustersResponse{
					Clusters: []*container.Cluster{
						&container.Cluster{
							Id:      "1",
							Zone:    "https://www.googleapis.com/compute/v1/projects/my_project/zones/europe-west1-d",
							Network: "my_network",
							Status:  "RUNNING",
						},
					},
				},
				"my_second_project": container.ListClustersResponse{
					Clusters: []*container.Cluster{
						&container.Cluster{
							Id:     "42",
							Zone:   "https://www.googleapis.com/compute/v1/projects/my_project/zones/us-central-1c",
							Status: "STOPPED",
						},
					},
				},
			},

			expectedClusters: []containerCluster{
				containerCluster{
					ID:      "1",
					Region:  "europe-west1",
					Account: "my_project",
					VPC:     "my_network",
					Metadata: mapstr.M{
						"state": "RUNNING",
					},
				},
				containerCluster{
					ID:      "42",
					Region:  "us-central",
					Account: "my_second_project",
					VPC:     "",
					Metadata: mapstr.M{
						"state": "STOPPED",
					},
				},
			},
		},
		{
			name: "with a regions filter",

			ctx: context.Background(),
			cfg: config{
				Projects: []string{"my_project"},
				Regions:  []string{"us-west1"},
			},

			httpResponses: map[string]container.ListClustersResponse{
				"my_project/us-west1": container.ListClustersResponse{
					Clusters: []*container.Cluster{
						&container.Cluster{
							Id:      "2",
							Zone:    "https://www.googleapis.com/compute/v1/projects/my_project/zones/us-west1-b",
							Network: "my_network",
							Status:  "RUNNING",
						},
					},
				},
			},

			expectedClusters: []containerCluster{
				containerCluster{
					ID:      "2",
					Region:  "us-west1",
					Account: "my_project",
					VPC:     "my_network",
					Metadata: mapstr.M{
						"state": "RUNNING",
					},
				},
			},
		},
		{
			name: "with multiple regions filters",

			ctx: context.Background(),
			cfg: config{
				Projects: []string{"my_project"},
				Regions:  []string{"us-west1,europe-west1"},
			},

			httpResponses: map[string]container.ListClustersResponse{
				"my_project/us-west1,europe-west1": container.ListClustersResponse{
					Clusters: []*container.Cluster{
						&container.Cluster{
							Id:      "2",
							Zone:    "https://www.googleapis.com/compute/v1/projects/my_project/zones/us-west1-b",
							Network: "my_network",
							Status:  "RUNNING",
						},
						&container.Cluster{
							Id:      "1",
							Zone:    "https://www.googleapis.com/compute/v1/projects/my_project/zones/europe-west1-d",
							Network: "my_network",
							Status:  "RUNNING",
						},
					},
				},
			},

			expectedClusters: []containerCluster{
				containerCluster{
					ID:      "2",
					Region:  "us-west1",
					Account: "my_project",
					VPC:     "my_network",
					Metadata: mapstr.M{
						"state": "RUNNING",
					},
				},
				containerCluster{
					ID:      "1",
					Region:  "europe-west1",
					Account: "my_project",
					VPC:     "my_network",
					Metadata: mapstr.M{
						"state": "RUNNING",
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				m := findGKEProjectRe.FindStringSubmatch(r.URL.Path)
				if len(m) < 2 {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				project := m[1]
				zone := m[2]
				path := project
				if zone != "-" {
					path = path + "/" + zone
				}

				b, err := json.Marshal(tt.httpResponses[path])
				assert.NoError(t, err)
				_, err = w.Write(b)
				assert.NoError(t, err)
			}))
			defer ts.Close()

			svc, err := container.NewService(tt.ctx, option.WithoutAuthentication(), option.WithEndpoint(ts.URL))
			assert.NoError(t, err)

			clusters, err := getAllGKEClusters(tt.ctx, tt.cfg, svc)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedClusters, clusters)
		})
	}
}
