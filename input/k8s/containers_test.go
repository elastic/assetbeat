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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/inputrunner/input/testutil"
)

func TestPublishK8sContainers(t *testing.T) {
	client := k8sfake.NewSimpleClientset()
	log := logp.NewLogger("mylogger")
	podWatcher, err := getPodWatcher(context.Background(), log, client, time.Second*60)
	if err != nil {
		t.Fatalf("error initiating Pod watcher")
	}
	containers := []v1.Container{
		{
			Name:  "nginx",
			Image: "nginx:1.120",
			Ports: []v1.ContainerPort{
				{
					Name:          "http",
					Protocol:      v1.ProtocolTCP,
					ContainerPort: 80,
				},
			},
		},
	}
	containerStatuses := []v1.ContainerStatus{
		{
			Name:        "nginx",
			Ready:       true,
			ContainerID: "crio://asdfghdeadbeef",
			State: v1.ContainerState{
				Running: &v1.ContainerStateRunning{},
			},
		},
	}
	input := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testpod",
			UID:       "a375d24b-fa20-4ea6-a0ee-1d38671d2c09",
			Namespace: "testns",
			Labels: map[string]string{
				"foo":        "bar",
				"with-dash":  "dash-value",
				"with/slash": "some/path",
			},
			Annotations: map[string]string{
				"app": "production",
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		Spec: v1.PodSpec{
			NodeName:   "testnode",
			Containers: containers,
		},
		Status: v1.PodStatus{
			PodIP:             "127.0.0.5",
			ContainerStatuses: containerStatuses,
		},
	}
	_ = podWatcher.Store().Add(input)
	publisher := testutil.NewInMemoryPublisher()
	publishK8sContainers(context.Background(), log, "", publisher, podWatcher)

	assert.Equal(t, 1, len(publisher.Events))
}
