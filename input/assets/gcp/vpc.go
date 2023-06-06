package gcp

import (
	"cloud.google.com/go/compute/apiv1/computepb"
	"context"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/inputrunner/input/assets/internal"
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/api/iterator"
	"strconv"
)

type NetworkIterator interface {
	Next() (*computepb.Network, error)
}

type listNetworkAPIClient struct {
	List func(ctx context.Context, req *computepb.ListNetworksRequest, opts ...gax.CallOption) NetworkIterator
}

type vpc struct {
	ID      string
	Name    string
	Account string
}

func collectNetworkAssets(ctx context.Context, cfg config, client listNetworkAPIClient, publisher stateless.Publisher) error {

	vpcs, err := getAllVPCs(ctx, cfg, client)

	if err != nil {
		return err
	}

	assetType := "gcp.vpc"
	assetKind := "network"
	indexNamespace := cfg.IndexNamespace
	for _, vpc := range vpcs {

		internal.Publish(publisher,
			internal.WithAssetCloudProvider("gcp"),
			internal.WithAssetAccountID(vpc.Account),
			internal.WithAssetTypeAndID(assetType, vpc.ID),
			internal.WithAssetName(vpc.Name),
			internal.WithAssetKind(assetKind),
			internal.WithIndex(assetType, indexNamespace),
		)
	}
	return nil
}

func getAllVPCs(ctx context.Context, cfg config, client listNetworkAPIClient) ([]vpc, error) {
	var vpcs []vpc
	for _, project := range cfg.Projects {
		req := &computepb.ListNetworksRequest{
			Project: project,
		}

		it := client.List(ctx, req)

		for {
			v, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, err
			}
			vpcs = append(vpcs, vpc{
				ID:      strconv.FormatUint(*v.Id, 10),
				Account: project,
				Name:    *v.Name,
			})
		}
	}
	return vpcs, nil

}
