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

package k8s

import (
	"context"
	"fmt"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/stretchr/testify/assert"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"testing"
	"time"
)

func Test_getNodeWatcher(t *testing.T) {
	client := k8sfake.NewSimpleClientset()
	log := logp.NewLogger("mylogger")
	_, err := getNodeWatcher(context.Background(), log, client, time.Second*60)
	if err != nil {
		t.Fatalf("error initiating Node watcher")
	}
	assert.NoError(t, err)
}

func Test_getNodeIdFromName(t *testing.T) {
	client := k8sfake.NewSimpleClientset()
	log := logp.NewLogger("mylogger")
	nodeWatcher, err := getNodeWatcher(context.Background(), log, client, time.Second*60)
	if err != nil {
		t.Fatalf("error initiating Node watcher")
	}

	_, err = getNodeIdFromName("nodeName", nodeWatcher)
	assert.Equal(t, err, fmt.Errorf("node with name %s does not exist in cache", "nodeName"))
}
