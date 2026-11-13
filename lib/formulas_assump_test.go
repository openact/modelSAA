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
	if err := categories.Add(1, "IG_Tradable", ""); err != nil {
		t.Fatal(err)
	}
	if err := categories.Add(2, "Cash", ""); err != nil {
		t.Fatal(err)
	}
	classes := enum.NewEnum("ASSET_CLASS")
	for i, value := range []string{"Cash", "Bond", "Equity", "Alternatives", "Others", "Dummy"} {
		if err := classes.Add(i+1, value, ""); err != nil {
			t.Fatal(err)
		}
	}
	bondTypes1 := enum.NewEnum("BOND_TYPE1")
	for i, value := range []string{"TRAD", "ALT", "NONE"} {
		if err := bondTypes1.Add(i+1, value, ""); err != nil {
			t.Fatal(err)
		}
	}
	return formulae.NewProjContext(&formulae.Setting{
		Global: &formulae.Global{},
		Enums:  map[string]*enum.Enum{"ASSET_CATEGORY": categories, "ASSET_CLASS": classes, "BOND_TYPE1": bondTypes1},
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
	ctx := assetMappingContext(t, "!2,ASSET_CATEGORY,ASSET_CLASS,BOND_TYPE1\n*,ig_tradable,bOnD,tRaD\n*,cash,cAsH,nOnE\n")
	if got := AssetClassByCategory(ctx, "IG_Tradable"); got != "Bond" {
		t.Fatalf("class = %s, want Bond", got)
	}
	if got := assetCategoryByValue(ctx, "Cash"); got != (assetCategorySpec{Category: "Cash", Class: "Cash", BondType1: "NONE"}) {
		t.Fatalf("spec = %+v", got)
	}
	if got := assetCategoryByValue(ctx, "IG_Tradable").BondType1; got != "TRAD" {
		t.Fatalf("bond type = %s, want TRAD", got)
	}
	delete(ctx.Setting.Tables, "assump")
	ctx.Cache.Tick(1)
	if got := AssetClassByCategory(ctx, "IG_Tradable"); got != "Bond" {
		t.Fatalf("cached class = %s, want Bond", got)
	}
	other := assetMappingContext(t, "!2,ASSET_CATEGORY,ASSET_CLASS,BOND_TYPE1\n*,IG_Tradable,Alternatives,ALT\n*,Cash,Cash,NONE\n")
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
		{"missing category", "!2,ASSET_CATEGORY,ASSET_CLASS,BOND_TYPE1\n*,IG_Tradable,Bond,TRAD\n", "missing ASSET_CATEGORY configuration: Cash"},
		{"unknown category", "!2,ASSET_CATEGORY,ASSET_CLASS,BOND_TYPE1\n*,Unknown,Bond,TRAD\n", `invalid ASSET_CATEGORY value "unknown"`},
		{"empty category", "!2,ASSET_CATEGORY,ASSET_CLASS,BOND_TYPE1\n*,,Bond,TRAD\n", `invalid ASSET_CATEGORY value ""`},
		{"empty class", "!2,ASSET_CATEGORY,ASSET_CLASS,BOND_TYPE1\n*,IG_Tradable,,TRAD\n*,Cash,Cash,NONE\n", `invalid ASSET_CLASS value ""`},
		{"unknown class", "!2,ASSET_CATEGORY,ASSET_CLASS,BOND_TYPE1\n*,IG_Tradable,Fixed_Income,TRAD\n*,Cash,Cash,NONE\n", `ASSET_CATEGORY "IG_Tradable": record 1, field Class: invalid ASSET_CLASS value "Fixed_Income"`},
		{"missing class column", "!2,ASSET_CATEGORY,CLASS,BOND_TYPE1\n*,IG_Tradable,Bond,TRAD\n*,Cash,Cash,NONE\n", "ASSET_CLASS"},
		{"missing bond type column", "!2,ASSET_CATEGORY,ASSET_CLASS\n*,IG_Tradable,Bond\n*,Cash,Cash\n", "BOND_TYPE1"},
		{"empty bond type", "!2,ASSET_CATEGORY,ASSET_CLASS,BOND_TYPE1\n*,IG_Tradable,Bond,\n*,Cash,Cash,NONE\n", `invalid BOND_TYPE1 value ""`},
		{"unknown bond type", "!2,ASSET_CATEGORY,ASSET_CLASS,BOND_TYPE1\n*,IG_Tradable,Bond,IG\n*,Cash,Cash,NONE\n", `invalid BOND_TYPE1 value "IG"`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := assetMappingContext(t, tt.content)
			expectAssetMappingPanic(t, tt.want, func() { loadAssetCategorySpecs(ctx) })
		})
	}
}

func TestLoadAssetCategorySpecsRequiresEnumsAndTable(t *testing.T) {
	const valid = "!2,ASSET_CATEGORY,ASSET_CLASS,BOND_TYPE1\n*,IG_Tradable,Bond,TRAD\n*,Cash,Cash,NONE\n"
	for _, name := range []string{"ASSET_CLASS", "ASSET_CATEGORY", "BOND_TYPE1"} {
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
			if err := e.Add(e.Size()+1, strings.ToLower(e.Values()[0]), ""); err != nil {
				t.Fatal(err)
			}
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
	content := "!2,ASSET_CATEGORY,ASSET_CLASS,BOND_TYPE1\n*,IG_Tradable,Bond,TRAD\n*,ig_tradable,Equity,NONE\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := cache.LoadTable(path, cache.LoadOptions{SuppressLoadLog: true}); err == nil ||
		!strings.Contains(err.Error(), "duplicate row key") {
		t.Fatalf("error = %v, want duplicate row key", err)
	}
}

func TestBondFormulasFollowConfiguredGroups(t *testing.T) {
	for _, tt := range []struct {
		name          string
		tradableType1 string
		cashType1     string
		wantTrad      float64
		wantAlt       float64
	}{
		{"original grouping", "TRAD", "NONE", 10, 0},
		{"reassigned categories", "ALT", "TRAD", 20, 10},
		{"combined group", "TRAD", "TRAD", 30, 0},
		{"no bond groups", "NONE", "NONE", 0, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			content := fmt.Sprintf("!2,ASSET_CATEGORY,ASSET_CLASS,BOND_TYPE1\n*,Cash,Cash,%s\n*,IG_Tradable,Bond,%s\n",
				tt.cashType1, tt.tradableType1)
			ctx := assetMappingContext(t, content)
			if err := NAR_FI.InitializeWithEnums(ctx.Setting); err != nil {
				t.Fatal(err)
			}
			ctx.ResultsStore.NumData[formulae.ExpandVarName("NAR_FI", []int{1})] = []float64{10, 100}
			ctx.ResultsStore.NumData[formulae.ExpandVarName("NAR_FI", []int{2})] = []float64{20, 200}
			for step, scale := range []float64{1, 10} {
				if got := BOND_TRAD.NumFunc(ctx, step, nil); got != tt.wantTrad*scale {
					t.Fatalf("BOND_TRAD[%d] = %g, want %g", step, got, tt.wantTrad*scale)
				}
				if got := BOND_ALT.NumFunc(ctx, step, nil); got != tt.wantAlt*scale {
					t.Fatalf("BOND_ALT[%d] = %g, want %g", step, got, tt.wantAlt*scale)
				}
			}
		})
	}
}
