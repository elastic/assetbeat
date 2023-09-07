package azure

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/elastic/assetbeat/input/internal"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/libbeat/feature"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/ctxtool"
	"time"
)

func Plugin() input.Plugin {
	return input.Plugin{
		Name:       "assets_azure",
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "assets_azure",
		Manager:    stateless.NewInputManager(configure),
	}
}

func configure(inputCfg *conf.C) (stateless.Input, error) {
	cfg := defaultConfig()
	if err := inputCfg.Unpack(&cfg); err != nil {
		return nil, err
	}

	return newAssetsAzure(cfg)
}

func newAssetsAzure(cfg config) (*assetsAzure, error) {
	return &assetsAzure{cfg}, nil
}

// Required credentials for the azure module:
//
// client_id
// The unique identifier for the application (also known as Application Id)
// client_secret
// The client/application secret/key
// subscription_id
// The unique identifier for the azure subscription
// tenant_id
// The unique identifier of the Azure Active Directory instance
//
// The azure credentials keys can be used if configured AZURE_CLIENT_ID, AZURE_CLIENT_SECRET, AZURE_TENANT_ID, AZURE_SUBSCRIPTION_ID
type config struct {
	internal.BaseConfig `config:",inline"`
	Regions             []string `config:"regions"`
	ClientID            string   `config:"client_id"`
	ClientSecret        string   `config:"client_secret"`
	SubscriptionID      string   `config:"subscription_id"`
	TenantID            string   `config:"tenant_id"`
}

func defaultConfig() config {
	return config{
		BaseConfig: internal.BaseConfig{
			Period:     time.Second * 600,
			AssetTypes: nil,
		},
		Regions:        []string{"westeurope"},
		ClientID:       "",
		ClientSecret:   "",
		SubscriptionID: "",
		TenantID:       "",
	}
}

type assetsAzure struct {
	Config config
}

func (s *assetsAzure) Name() string { return "assets_azure" }

func (s *assetsAzure) Test(_ input.TestContext) error {
	return nil
}

func (s *assetsAzure) Run(inputCtx input.Context, publisher stateless.Publisher) error {
	ctx := ctxtool.FromCanceller(inputCtx.Cancelation)
	log := inputCtx.Logger.With("assets_azure")

	log.Info("azure asset collector run started")
	defer log.Info("azure asset collector run stopped")

	cfg := s.Config
	period := cfg.Period

	ticker := time.NewTicker(period)
	select {
	case <-ctx.Done():
		return nil
	default:
		collectAzureAssets(ctx, log, cfg, publisher)
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			collectAzureAssets(ctx, log, cfg, publisher)
		}
	}
}

func collectAzureAssets(ctx context.Context, log *logp.Logger, cfg config, publisher stateless.Publisher) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Errorf("Error retrieving Azure creds: %v", err)
	}

	err = collectAzureVMAssets(ctx, cfg, cred, log, publisher)
	if err != nil {
		log.Errorf("Error while collecting Azure VM assets: %v", err)
	}

}
