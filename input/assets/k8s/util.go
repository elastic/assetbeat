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
	default:
		return ""
	}
	return ""
}

// getCspFromProviderId return the cps for a given providerId string.
// In case of aws providerId is in the form of aws:///region/instanceId
// In case of gcp providerId is in the form of  gce://project/region/nodeName
func getCspFromProviderId(providerId string) string {
	if strings.HasPrefix(providerId, "aws") {
		return "aws"
	}
	if strings.HasPrefix(providerId, "gce") {
		return "gcp"
	}
	return ""
}

// getGKEClusterUid gets the GKE cluster uid from metadata endpoint
func getGKEClusterUid(ctx context.Context, log *logp.Logger) (string, error) {
	url := fmt.Sprintf("http://%s%s", metadataHost, gceMetadataURI)
	gceHeaders := map[string]string{"Metadata-Flavor": "Google"}
	// Create HTTP client with our timeouts and keep-alive disabled.
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

	response, err := fetchHttpResponse(ctx, url, client, gceHeaders)
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

// fetchHttpResponse returns http response of a provided URL
func fetchHttpResponse(ctx context.Context, url string, client http.Client, headers map[string]string) ([]byte, error) {
	var response []byte
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return response, errors.Wrapf(err, "failed to create http request for gcp")
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	req = req.WithContext(ctx)

	rsp, err := client.Do(req)
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
