package lib

import (
	"errors"
	"fmt"

	"github.com/openact/formulae"
	"github.com/openact/kit/cache/v3"
	"github.com/openact/kit/enum"
)

type assetCategorySpec struct {
	Category  string `rowkey:"ASSET_CATEGORY" enum:"ASSET_CATEGORY"`
	Class     string `colkey:"ASSET_CLASS" enum:"ASSET_CLASS"`
	BondType1 string `colkey:"BOND_TYPE1" enum:"BOND_TYPE1"`
}

type assetCategorySpecs struct {
	byCategory    map[string]assetCategorySpec
	categoryOrder []string
}

func AssetClassByCategory(ctx *formulae.ProjContext, category string) string {
	return assetCategoryByValue(ctx, category).Class
}

func assetCategoryByValue(ctx *formulae.ProjContext, category string) assetCategorySpec {
	spec, ok := getAssetCategorySpecs(ctx).byCategory[category]
	if !ok {
		panic("unknown ASSET_CATEGORY: " + category)
	}
	return spec
}

func getAssetCategorySpecs(ctx *formulae.ProjContext) *assetCategorySpecs {
	return formulae.GetOrInitCache(ctx, "assetCategorySpecs", func() *assetCategorySpecs {
		return loadAssetCategorySpecs(ctx)
	})
}

func loadAssetCategorySpecs(ctx *formulae.ProjContext) *assetCategorySpecs {
	table := ctx.Table("assump")
	specs, err := cache.ScanRows[assetCategorySpec](table)
	if err != nil {
		panic(err)
	}
	if err := enum.NormalizeFields(specs, ctx.Setting.Enums); err != nil {
		var fieldErr *enum.FieldError
		if errors.As(err, &fieldErr) {
			panic(fmt.Errorf("%s, ASSET_CATEGORY %q: %w", table.FilePath, specs[fieldErr.Row].Category, err))
		}
		panic(fmt.Errorf("%s: %w", table.FilePath, err))
	}
	byCategory := make(map[string]assetCategorySpec, len(specs))
	for _, spec := range specs {
		if _, exists := byCategory[spec.Category]; exists {
			panic("duplicate ASSET_CATEGORY configuration: " + spec.Category)
		}
		byCategory[spec.Category] = spec
	}
	categoryOrder := ctx.GetEnum("ASSET_CATEGORY").Values()
	for _, category := range categoryOrder {
		if _, ok := byCategory[category]; !ok {
			panic("missing ASSET_CATEGORY configuration: " + category)
		}
	}
	return &assetCategorySpecs{byCategory: byCategory, categoryOrder: categoryOrder}
}
