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
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/elastic-agent-autodiscover/kubernetes"
	"github.com/elastic/elastic-agent-libs/logp"
)

type httpResponse interface {
	FetchResponse(context.Context, string, map[string]string) ([]byte, error)
}

// httpFetcher struct to implement httpResponse interface
type httpFetcher struct {
	httpClient http.Client
}

// newhttpFetcher returns a new httpFetcher
func newhttpFetcher() httpFetcher {
	client := http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			DialContext: (&net.Dialer{
				Timeout:   time.Second * 10,
				KeepAlive: 0,
			}).DialContext,
		},
	}
	return httpFetcher{httpClient: client}
}

// getInstanceId returns the cloud instance id in case
// the node runs in one of [aws, gcp] csp.
// In case of aws the instance id is retrieved from providerId
// which is in the form of aws:///region/instanceId for not fargate nodes.
// In case of gcp it is retrieved by the annotation container.googleapis.com/instance_id
// In all other cases empty string is returned
func getInstanceId(node *kubernetes.Node) string {
	providerId := node.Spec.ProviderID

	switch csp := getCspFromProviderId(providerId); csp {
	case "aws":
		slice := strings.Split(providerId, "/")
		// in case of fargate the slice length will be 6
		if len(slice) == 5 {
			return slice[4]
		}
	case "gcp":
		annotations := node.GetAnnotations()
		return annotations["container.googleapis.com/instance_id"]
	case "azure":
		return node.Status.NodeInfo.SystemUUID
	default:
		return ""
	}
	return ""
}

// getNodeState returns the state of kubernets node
// the state can be Ready, MemoryPressure, DiskPressure, PIDPressure,
// NetworkUnavailable or Unknown
func getNodeState(node *kubernetes.Node) string {
	conditions := node.Status.Conditions
	status := "Unknown"
	for _, condition := range conditions {
		if condition.Status == "True" {
			status = string(condition.Type)
		}
	}
	return status
}

// getCspFromProviderId return the cps for a given providerId string.
// In case of aws providerId is in the form of aws:///region/instanceId
// In case of gcp providerId is in the form of  gce://project/region/nodeName
// In case of azure providerId is in the form of  azure:///subscriptions/subscriptionId
func getCspFromProviderId(providerId string) string {
	if strings.HasPrefix(providerId, "aws") {
		return "aws"
	}
	if strings.HasPrefix(providerId, "gce") {
		return "gcp"
	}
	if strings.HasPrefix(providerId, "azure") {
		return "azure"
	}
	return ""
}

// getGKEClusterUid gets the GKE cluster uid from metadata endpoint
func getGKEClusterUid(ctx context.Context, log *logp.Logger, hF httpResponse) (string, error) {
	url := fmt.Sprintf("http://%s%s", metadataHost, gceMetadataURI)
	gceHeaders := map[string]string{"Metadata-Flavor": "Google"}

	response, err := hF.FetchResponse(ctx, url, gceHeaders)
	if err != nil {
		return "", err
	}
	var gcpMetadataRes = map[string]interface{}{}
	if err = json.Unmarshal(response, &gcpMetadataRes); err != nil {
		return "", err
	}
	if instance, ok := gcpMetadataRes["instance"].(map[string]interface{}); ok {
		if attributes, ok := instance["attributes"].(map[string]interface{}); ok {
			if clusterUid, ok := attributes["cluster-uid"].(string); ok {
				log.Debugf("Cloud uid is %s ", clusterUid)
				return clusterUid, nil
			}
		}
	}
	return "", errors.Errorf("Cluster uid not found in metadata")
}

// FetchResponse returns http response of a provided URL
func (c httpFetcher) FetchResponse(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	var response []byte
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return response, errors.Wrapf(err, "failed to create http request for gcp")
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	req = req.WithContext(ctx)

	rsp, err := c.httpClient.Do(req)
	if err != nil {
		return response, errors.Wrapf(err, "failed requesting gcp metadata")
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		return response, errors.Errorf("failed with http status code %v", rsp.StatusCode)
	}

	response, err = io.ReadAll(rsp.Body)
	if err != nil {
		return response, errors.Wrapf(err, "failed requesting gcp metadata")
	}
	return response, nil
}
