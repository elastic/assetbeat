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

package aws

import (
	"context"
	"testing"
	"time"

	"github.com/elastic/inputrunner/input/assets/internal"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
)

func TestGetAWSConfigForRegion(t *testing.T) {
	for _, tt := range []struct {
		name string

		env           map[string]string
		inputCfg      config
		region        string
		expectedCreds aws.Credentials
	}{
		{
			name: "with explicit creds",

			inputCfg: config{
				BaseConfig: internal.BaseConfig{
					Period:     time.Second * 600,
					AssetTypes: []string{},
				},
				Regions:         []string{"eu-west-2", "eu-west-1"},
				AccessKeyId:     "accesskey123",
				SecretAccessKey: "secretkey123",
				SessionToken:    "token123",
			},
			region: "eu-west-2",

			expectedCreds: aws.Credentials{
				AccessKeyID:     "accesskey123",
				SecretAccessKey: "secretkey123",
				SessionToken:    "token123",
				Source:          "inputrunner configuration",
			},
		},
		{
			name: "with environment variable creds",

			env: map[string]string{
				"AWS_ACCESS_KEY":        "EXAMPLE_ACCESS_KEY",
				"AWS_SECRET_ACCESS_KEY": "EXAMPLE_SECRET_KEY",
			},
			inputCfg: config{
				BaseConfig: internal.BaseConfig{
					Period:     time.Second * 600,
					AssetTypes: []string{},
				},
				Regions:         []string{"eu-west-2", "eu-west-1"},
				AccessKeyId:     "",
				SecretAccessKey: "",
				SessionToken:    "",
			},
			region: "eu-west-2",

			expectedCreds: aws.Credentials{
				AccessKeyID:     "EXAMPLE_ACCESS_KEY",
				SecretAccessKey: "EXAMPLE_SECRET_KEY",
				SessionToken:    "",
				Source:          "EnvConfigCredentials",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			awsCfg, err := getAWSConfigForRegion(ctx, tt.inputCfg, tt.region)
			assert.NoError(t, err)

			retrievedAWSCreds, err := awsCfg.Credentials.Retrieve(context.Background())
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCreds, retrievedAWSCreds)
		})
	}
}
