package aws

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	typesAsg "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/stretchr/testify/assert"
	"testing"
)

const clusterArnPrefix = "arn:aws:eks:eu-west-1:12345678:cluster/"

var instanceID1 = "i-1111111"
var instanceID2 = "i-2222222"

type mockDescribeClusterAPI func(ctx context.Context, params *eks.DescribeClusterInput, optFns ...func(*eks.Options)) (*eks.DescribeClusterOutput, error)

func (m mockDescribeClusterAPI) DescribeCluster(ctx context.Context, params *eks.DescribeClusterInput, optFns ...func(*eks.Options)) (*eks.DescribeClusterOutput, error) {
	return m(ctx, params, optFns...)
}

type mockListNodeGroupsAPI func(ctx context.Context, params *eks.ListNodegroupsInput, optFns ...func(*eks.Options)) (*eks.ListNodegroupsOutput, error)

func (m mockListNodeGroupsAPI) ListNodegroups(ctx context.Context, params *eks.ListNodegroupsInput, optFns ...func(*eks.Options)) (*eks.ListNodegroupsOutput, error) {
	return m(ctx, params, optFns...)
}

type mockDescribeNodeGroupsAPI func(ctx context.Context, params *eks.DescribeNodegroupInput, optFns ...func(*eks.Options)) (*eks.DescribeNodegroupOutput, error)

func (m mockDescribeNodeGroupsAPI) DescribeNodegroup(ctx context.Context, params *eks.DescribeNodegroupInput, optFns ...func(*eks.Options)) (*eks.DescribeNodegroupOutput, error) {
	return m(ctx, params, optFns...)
}

type mockDescribeAutoscalingGroupsAPI func(ctx context.Context, params *autoscaling.DescribeAutoScalingGroupsInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error)

func (m mockDescribeAutoscalingGroupsAPI) DescribeAutoScalingGroups(ctx context.Context, params *autoscaling.DescribeAutoScalingGroupsInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
	return m(ctx, params, optFns...)
}

func TestDescribeEKSClusters(t *testing.T) {
	for _, tt := range []struct {
		name     string
		ctx      context.Context
		log      *logp.Logger
		clusters []string
		client   func(t *testing.T) eks.DescribeClusterAPIClient
	}{
		{
			name:     "test",
			ctx:      context.Background(),
			log:      logp.NewLogger("test"),
			clusters: []string{"test-cluster1", "test-cluster2"},
			client: func(t *testing.T) eks.DescribeClusterAPIClient {
				return mockDescribeClusterAPI(func(ctx context.Context, params *eks.DescribeClusterInput, optFns ...func(*eks.Options)) (*eks.DescribeClusterOutput, error) {
					t.Helper()
					arn := clusterArnPrefix + (*params.Name)
					return &eks.DescribeClusterOutput{
						Cluster: &types.Cluster{
							Arn:  &arn,
							Name: params.Name,
						},
						ResultMetadata: middleware.Metadata{},
					}, nil
				})
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			clusters := describeEKSClusters(tt.log, tt.ctx, tt.clusters, tt.client(t))
			assert.NotNil(t, clusters)
			assert.Len(t, clusters, len(tt.clusters))
			for i, cluster := range clusters {
				assert.Equal(t, *cluster.Name, tt.clusters[i])
				assert.Equal(t, *cluster.Arn, clusterArnPrefix+tt.clusters[i])
			}
		})
	}
}

func TestListNodeGroups(t *testing.T) {
	for _, tt := range []struct {
		name               string
		ctx                context.Context
		log                *logp.Logger
		cluster            string
		client             func(t *testing.T) eks.ListNodegroupsAPIClient
		expectedNodeGroups []string
	}{
		{
			name:               "test",
			ctx:                context.Background(),
			log:                logp.NewLogger("test"),
			cluster:            "test-cluster",
			expectedNodeGroups: []string{"test-nodegroup"},
			client: func(t *testing.T) eks.ListNodegroupsAPIClient {
				return mockListNodeGroupsAPI(func(ctx context.Context, params *eks.ListNodegroupsInput, optFns ...func(*eks.Options)) (*eks.ListNodegroupsOutput, error) {
					t.Helper()
					return &eks.ListNodegroupsOutput{
						NextToken:      nil,
						Nodegroups:     []string{"test-nodegroup"},
						ResultMetadata: middleware.Metadata{},
					}, nil
				})
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			nodeGroups, err := listNodeGroups(tt.ctx, tt.cluster, tt.client(t))
			assert.NoError(t, err)
			assert.NotNil(t, nodeGroups)
			assert.Equal(t, nodeGroups, tt.expectedNodeGroups)
		})
	}
}

func TestGetInstanceIDsFromNodeGroup(t *testing.T) {
	for _, tt := range []struct {
		name        string
		ctx         context.Context
		log         *logp.Logger
		cluster     string
		eksClient   func(t *testing.T) eks.DescribeNodegroupAPIClient
		asgClient   func(t *testing.T) autoscaling.DescribeAutoScalingGroupsAPIClient
		asgGroup    string
		instanceIDs []string
		nodeGroups  []string
	}{
		{
			name:        "test",
			ctx:         context.Background(),
			log:         logp.NewLogger("test"),
			cluster:     "test-cluster",
			nodeGroups:  []string{"test-nodegroup"},
			asgGroup:    "test-asg",
			instanceIDs: []string{instanceID1, instanceID2},
			eksClient: func(t *testing.T) eks.DescribeNodegroupAPIClient {
				return mockDescribeNodeGroupsAPI(func(ctx context.Context, params *eks.DescribeNodegroupInput, optFns ...func(*eks.Options)) (*eks.DescribeNodegroupOutput, error) {
					t.Helper()
					asg := "test-asg"
					return &eks.DescribeNodegroupOutput{
						Nodegroup: &types.Nodegroup{
							NodegroupName: params.NodegroupName,
							Resources: &types.NodegroupResources{
								AutoScalingGroups: []types.AutoScalingGroup{
									{Name: &asg},
								},
							},
						},
						ResultMetadata: middleware.Metadata{},
					}, nil
				})
			},
			asgClient: func(t *testing.T) autoscaling.DescribeAutoScalingGroupsAPIClient {
				return mockDescribeAutoscalingGroupsAPI(func(ctx context.Context, params *autoscaling.DescribeAutoScalingGroupsInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
					t.Helper()
					return &autoscaling.DescribeAutoScalingGroupsOutput{
						AutoScalingGroups: []typesAsg.AutoScalingGroup{
							{
								Instances: []typesAsg.Instance{
									{
										InstanceId: &instanceID1,
									},
									{
										InstanceId: &instanceID2,
									},
								},
							},
						},
						NextToken:      nil,
						ResultMetadata: middleware.Metadata{},
					}, nil
				})
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			instances, err := getInstanceIDsFromNodeGroup(tt.ctx, tt.cluster, tt.nodeGroups, tt.eksClient(t), tt.asgClient(t))
			assert.NoError(t, err)
			assert.Equal(t, instances, tt.instanceIDs)
		})
	}
}
