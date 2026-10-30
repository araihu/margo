package site

import (
	"fmt"

	margo "github.com/araihu/margo"
)

const maxSiteAssetBudget int64 = 256 << 20

func validateAssetBudget(value int64) error {
	if value < 0 || value > maxSiteAssetBudget {
		return diagnostic("site.asset_budget_invalid", fmt.Sprintf("asset_budget_bytes must be between 0 and %d", maxSiteAssetBudget), "Use zero for the default 16 MiB aggregate or an explicit bounded byte count.", "")
	}
	return nil
}

func (b *builder) sourceAssetLimit() int64 {
	budget := b.request.AssetBudgetBytes
	if budget == 0 {
		budget = margo.MaxDocumentBytes
	}
	return max(0, min(margo.MaxDocumentBytes, budget-b.assetBytes))
}
