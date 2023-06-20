package gcp

import "sync"

type VpcAssetsCache struct {
	vpcAssets map[string]*vpc
	lock      sync.Mutex
}

func (c *VpcAssetsCache) getAssetEntry(selfLink string) *vpc {
	if _, ok := c.vpcAssets[selfLink]; ok {
		return c.vpcAssets[selfLink]
	}
	return nil
}

func (c *VpcAssetsCache) setAssetEntry(selfLink string, vpc *vpc) {
	c.vpcAssets[selfLink] = vpc
}
