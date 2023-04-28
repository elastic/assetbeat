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

package inputs

import (
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/inputrunner/beater"
	"github.com/elastic/inputrunner/input/assets/aws"
	"github.com/elastic/inputrunner/input/assets/gcp"
	"github.com/elastic/inputrunner/input/assets/k8s"
	"github.com/elastic/inputrunner/input/exec"
	"github.com/elastic/inputrunner/input/udp"
	"github.com/elastic/inputrunner/input/unix"
)

func Init(info beat.Info, log *logp.Logger, components beater.StateStore) []v2.Plugin {
	return genericInputs(log, components)
}

func genericInputs(log *logp.Logger, components beater.StateStore) []v2.Plugin {
	return []v2.Plugin{
		aws.Plugin(),
		k8s.Plugin(),
		gcp.Plugin(),
		exec.Plugin(),
		udp.Plugin(),
		unix.Plugin(),
	}
}
