package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openact/formulae"
	"github.com/openact/kit/cache/v3"
	"github.com/openact/kit/enum"
)

func assetMappingContext(t *testing.T, content string) *formulae.ProjContext {
	t.Helper()
	path := filepath.Join(t.TempDir(), "Assump.fac")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	table, err := cache.LoadTable(path, cache.LoadOptions{SuppressLoadLog: true})
	if err != nil {
		t.Fatal(err)
	}
	categories := enum.NewEnum("ASSET_CATEGORY")
	categories.Add(1, "IG_Tradable", "")
	categories.Add(2, "Cash", "")
	classes := enum.NewEnum("ASSET_CLASS")
	for i, value := range []string{"Cash", "Bond", "Equity", "Alternatives", "Others", "Dummy"} {
		classes.Add(i+1, value, "")
	}
	return formulae.NewProjContext(&formulae.Setting{
		Global: &formulae.Global{},
		Enums:  map[string]*enum.Enum{"ASSET_CATEGORY": categories, "ASSET_CLASS": classes},
		Tables: formulae.Tables{"assump": table},
	}, "1", "ALM", &formulae.ModelPoint{})
}

func expectAssetMappingPanic(t *testing.T, want string, fn func()) {
	t.Helper()
	defer func() {
		if got := recover(); got == nil || !strings.Contains(fmt.Sprint(got), want) {
			t.Fatalf("panic = %v, want %q", got, want)
		}
	}()
	fn()
}

func TestAssetClassByCategoryCanonicalizesAndCaches(t *testing.T) {
	ctx := assetMappingContext(t, "!2,ASSET_CATEGORY,ASSET_CLASS\n*,ig_tradable,bOnD\n*,cash,cAsH\n")
	if got := AssetClassByCategory(ctx, "IG_Tradable"); got != "Bond" {
		t.Fatalf("class = %s, want Bond", got)
	}
	if got := assetCategoryByValue(ctx, "Cash"); got != (assetCategorySpec{Category: "Cash", Class: "Cash"}) {
		t.Fatalf("spec = %+v", got)
	}
	delete(ctx.Setting.Tables, "assump")
	ctx.Cache.Tick(1)
	if got := AssetClassByCategory(ctx, "IG_Tradable"); got != "Bond" {
		t.Fatalf("cached class = %s, want Bond", got)
	}
	other := assetMappingContext(t, "!2,ASSET_CATEGORY,ASSET_CLASS\n*,IG_Tradable,Alternatives\n*,Cash,Cash\n")
	if got := AssetClassByCategory(other, "IG_Tradable"); got != "Alternatives" {
		t.Fatalf("independent context class = %s, want Alternatives", got)
	}
	expectAssetMappingPanic(t, "unknown ASSET_CATEGORY: Unknown", func() {
		AssetClassByCategory(ctx, "Unknown")
	})
}

func TestLoadAssetCategorySpecsRejectsInvalidMappings(t *testing.T) {
	for _, tt := range []struct {
		name    string
		content string
		want    string
	}{
		{"missing category", "!2,ASSET_CATEGORY,ASSET_CLASS\n*,IG_Tradable,Bond\n", "missing ASSET_CATEGORY configuration: Cash"},
		{"unknown category", "!2,ASSET_CATEGORY,ASSET_CLASS\n*,Unknown,Bond\n", "unknown ASSET_CATEGORY configuration: unknown"},
		{"empty category", "!2,ASSET_CATEGORY,ASSET_CLASS\n*,,Bond\n", "unknown ASSET_CATEGORY configuration:"},
		{"empty class", "!2,ASSET_CATEGORY,ASSET_CLASS\n*,IG_Tradable,\n*,Cash,Cash\n", "invalid ASSET_CLASS for IG_Tradable:"},
		{"unknown class", "!2,ASSET_CATEGORY,ASSET_CLASS\n*,IG_Tradable,Fixed_Income\n*,Cash,Cash\n", "invalid ASSET_CLASS for IG_Tradable: Fixed_Income"},
		{"missing column", "!2,ASSET_CATEGORY,CLASS\n*,IG_Tradable,Bond\n*,Cash,Cash\n", "ASSET_CLASS"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := assetMappingContext(t, tt.content)
			expectAssetMappingPanic(t, tt.want, func() { loadAssetCategorySpecs(ctx) })
		})
	}
}

func TestLoadAssetCategorySpecsRequiresEnumsAndTable(t *testing.T) {
	const valid = "!2,ASSET_CATEGORY,ASSET_CLASS\n*,IG_Tradable,Bond\n*,Cash,Cash\n"
	for _, name := range []string{"ASSET_CLASS", "ASSET_CATEGORY"} {
		t.Run("missing "+name, func(t *testing.T) {
			ctx := assetMappingContext(t, valid)
			delete(ctx.Setting.Enums, name)
			expectAssetMappingPanic(t, "missing enum: "+name, func() { loadAssetCategorySpecs(ctx) })
		})
		t.Run("empty "+name, func(t *testing.T) {
			ctx := assetMappingContext(t, valid)
			ctx.Setting.Enums[name] = enum.NewEnum(name)
			expectAssetMappingPanic(t, "empty enum: "+name, func() { loadAssetCategorySpecs(ctx) })
		})
		t.Run("ambiguous "+name, func(t *testing.T) {
			ctx := assetMappingContext(t, valid)
			e := ctx.Setting.Enums[name]
			e.Add(e.Size()+1, strings.ToLower(e.Values()[0]), "")
			expectAssetMappingPanic(t, "ambiguous enum values in "+name, func() { loadAssetCategorySpecs(ctx) })
		})
	}
	t.Run("missing table", func(t *testing.T) {
		ctx := assetMappingContext(t, valid)
		delete(ctx.Setting.Tables, "assump")
		expectAssetMappingPanic(t, "assump", func() { loadAssetCategorySpecs(ctx) })
	})
}

func TestAssumpTableRejectsDuplicateCategory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Assump.fac")
	content := "!2,ASSET_CATEGORY,ASSET_CLASS\n*,IG_Tradable,Bond\n*,ig_tradable,Equity\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.LoadTable(path, cache.LoadOptions{SuppressLoadLog: true}); err == nil ||
		!strings.Contains(err.Error(), "duplicate row key") {
		t.Fatalf("error = %v, want duplicate row key", err)
	}
}
