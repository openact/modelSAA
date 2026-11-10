package lib

import (
	"strings"

	"github.com/openact/formulae"
	"github.com/openact/kit/cache/v3"
)

type assetCategorySpec struct {
	Category string `rowkey:"ASSET_CATEGORY"`
	Class    string `colkey:"ASSET_CLASS"`
}

func AssetClassByCategory(ctx *formulae.ProjContext, category string) string {
	return assetCategoryByValue(ctx, category).Class
}

func assetCategoryByValue(ctx *formulae.ProjContext, category string) assetCategorySpec {
	spec, ok := getAssetCategorySpecs(ctx)[category]
	if !ok {
		panic("unknown ASSET_CATEGORY: " + category)
	}
	return spec
}

func getAssetCategorySpecs(ctx *formulae.ProjContext) map[string]assetCategorySpec {
	return formulae.GetOrInitCache(ctx, "assetCategorySpecs", func() map[string]assetCategorySpec {
		return loadAssetCategorySpecs(ctx)
	})
}

func loadAssetCategorySpecs(ctx *formulae.ProjContext) map[string]assetCategorySpec {
	categories := assetEnumValues(ctx, "ASSET_CATEGORY")
	classes := assetEnumValues(ctx, "ASSET_CLASS")
	specs, err := cache.ScanRows[assetCategorySpec](ctx.Table("assump"))
	if err != nil {
		panic(err)
	}
	byCategory := make(map[string]assetCategorySpec, len(specs))
	for _, spec := range specs {
		category, ok := categories[strings.ToLower(spec.Category)]
		if !ok {
			panic("unknown ASSET_CATEGORY configuration: " + spec.Category)
		}
		if _, exists := byCategory[category]; exists {
			panic("duplicate ASSET_CATEGORY configuration: " + category)
		}
		class, ok := classes[strings.ToLower(spec.Class)]
		if !ok {
			panic("invalid ASSET_CLASS for " + category + ": " + spec.Class)
		}
		byCategory[category] = assetCategorySpec{Category: category, Class: class}
	}
	for _, category := range ctx.GetEnum("ASSET_CATEGORY").Values() {
		if _, ok := byCategory[category]; !ok {
			panic("missing ASSET_CATEGORY configuration: " + category)
		}
	}
	return byCategory
}

func assetEnumValues(ctx *formulae.ProjContext, name string) map[string]string {
	e := ctx.GetEnum(name)
	if e == nil {
		panic("missing enum: " + name)
	}
	if e.Size() == 0 {
		panic("empty enum: " + name)
	}
	values := make(map[string]string, e.Size())
	for _, value := range e.Values() {
		if value == "" {
			panic("empty enum value in " + name)
		}
		key := strings.ToLower(value)
		if existing, ok := values[key]; ok {
			panic("ambiguous enum values in " + name + ": " + existing + ", " + value)
		}
		values[key] = value
	}
	return values
}
