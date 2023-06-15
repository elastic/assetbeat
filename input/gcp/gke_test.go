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
	"fmt"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/googleapis/gax-go/v2"
	"regexp"
	"testing"

	"cloud.google.com/go/container/apiv1/containerpb"
	"github.com/stretchr/testify/assert"
)

type ClustersClientStub struct {
	Clusters map[string]*containerpb.ListClustersResponse
}

func (s *ClustersClientStub) ListClusters(ctx context.Context, req *containerpb.ListClustersRequest, opts ...gax.CallOption) (*containerpb.ListClustersResponse, error) {
	m := findGKEProjectRe.FindStringSubmatch(req.Parent)
	fmt.Println(req.Parent)
	project := m[1]
	fmt.Println(project)
	return s.Clusters[project], nil
}

var findGKEProjectRe = regexp.MustCompile("projects/([a-z_-]+)/locations/([0-9a-z_,-]+)")

func TestGetAllGKEClusters(t *testing.T) {
	for _, tt := range []struct {
		name string

		ctx          context.Context
		cfg          config
		apiResponses map[string]*containerpb.ListClustersResponse

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

			apiResponses: map[string]*containerpb.ListClustersResponse{
				"my_project": {
					Clusters: []*containerpb.Cluster{
						{
							Id:       "1",
							Location: "europe-west1",
							Network:  "my_network",
							Status:   containerpb.Cluster_RUNNING,
						},
					},
				},
			},

			expectedClusters: []containerCluster{
				{
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

			apiResponses: map[string]*containerpb.ListClustersResponse{
				"my_project": {
					Clusters: []*containerpb.Cluster{
						{
							Id:       "1",
							Location: "europe-west1",
							Network:  "my_network",
							Status:   containerpb.Cluster_RUNNING,
						},
					},
				},
				"my_second_project": {
					Clusters: []*containerpb.Cluster{
						{
							Id:       "42",
							Location: "us-central",
							Network:  "",
							Status:   containerpb.Cluster_STOPPING,
						},
					},
				},
			},

			expectedClusters: []containerCluster{
				{
					ID:      "1",
					Region:  "europe-west1",
					Account: "my_project",
					VPC:     "my_network",
					Metadata: mapstr.M{
						"state": "RUNNING",
					},
				},
				{
					ID:      "42",
					Region:  "us-central",
					Account: "my_second_project",
					VPC:     "",
					Metadata: mapstr.M{
						"state": "STOPPING",
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

			apiResponses: map[string]*containerpb.ListClustersResponse{
				"my_project": {
					Clusters: []*containerpb.Cluster{
						{
							Id:       "2",
							Location: "us-west1",
							Network:  "my_network",
							Status:   containerpb.Cluster_RUNNING,
						},
					},
				},
			},

			expectedClusters: []containerCluster{
				{
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

			apiResponses: map[string]*containerpb.ListClustersResponse{
				"my_project": {
					Clusters: []*containerpb.Cluster{
						{
							Id:       "2",
							Location: "us-west1",
							Network:  "my_network",
							Status:   containerpb.Cluster_RUNNING,
						},
						{
							Id:       "1",
							Location: "europe-west1",
							Network:  "my_network",
							Status:   containerpb.Cluster_RUNNING,
						},
					},
				},
			},

			expectedClusters: []containerCluster{
				{
					ID:      "2",
					Region:  "us-west1",
					Account: "my_project",
					VPC:     "my_network",
					Metadata: mapstr.M{
						"state": "RUNNING",
					},
				},
				{
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

			client := ClustersClientStub{Clusters: tt.apiResponses}
			clusters, err := getAllGKEClusters(tt.ctx, tt.cfg, &client)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedClusters, clusters)
		})
	}
}
