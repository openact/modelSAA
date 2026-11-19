package lib

import (
	"testing"

	"github.com/openact/formulae"
)

func TestSumWhereByAssetCategory(t *testing.T) {
	ctx := assetMappingContext(t, "!2,ASSET_CATEGORY,ASSET_CLASS,BOND_TYPE1\n*,Cash,Cash,TRAD\n*,IG_Tradable,Bond,TRAD\n")
	if err := ASSET_BAL.InitializeWithEnums(ctx.Setting); err != nil {
		t.Fatal(err)
	}
	if err := ctx.ResultsStore.BindDefinitions(ASSET_BAL); err != nil {
		t.Fatal(err)
	}
	ctx.ResultsStore.NumData[formulae.ExpandVarName("ASSET_BAL", []int{1})] = []float64{7, 11}
	ctx.ResultsStore.NumData[formulae.ExpandVarName("ASSET_BAL", []int{2})] = []float64{13, 17}
	for _, tt := range []struct {
		name    string
		include func(string) bool
		step    int
		want    float64
	}{
		{"empty group", func(category string) bool { return ctx.ReadTableTxt("Assump", "Y", category, "BOND_TYPE1") == "ALT" }, 0, 0},
		{"by class", func(category string) bool { return ctx.ReadTableTxt("Assump", "Y", category, "ASSET_CLASS") == "Cash" }, 0, 13},
		{"by category", func(category string) bool { return category == "IG_Tradable" }, 0, 7},
		{"by bond type1", func(category string) bool { return ctx.ReadTableTxt("Assump", "Y", category, "BOND_TYPE1") == "TRAD" }, 0, 20},
		{"next step", func(category string) bool { return ctx.ReadTableTxt("Assump", "Y", category, "BOND_TYPE1") == "TRAD" }, 1, 28},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := ctx.ResultsStore.SumWhere("ASSET_BAL", tt.step, "ASSET_CATEGORY", tt.include); got != tt.want {
				t.Fatalf("sum at step %d = %g, want %g", tt.step, got, tt.want)
			}
		})
	}
}

func TestSumWhereByAssetCategoryPreservesEnumOrder(t *testing.T) {
	ctx := assetMappingContext(t, "!2,ASSET_CATEGORY,ASSET_CLASS,BOND_TYPE1\n*,Cash,Cash,TRAD\n*,Other_Asset,Others,TRAD\n*,IG_Tradable,Bond,TRAD\n")
	if err := ctx.Setting.Enums["ASSET_CATEGORY"].Add(3, "Other_Asset", ""); err != nil {
		t.Fatal(err)
	}
	if err := NAR_TOT.InitializeWithEnums(ctx.Setting); err != nil {
		t.Fatal(err)
	}
	if err := ctx.ResultsStore.BindDefinitions(NAR_TOT); err != nil {
		t.Fatal(err)
	}
	for key, value := range map[int]float64{1: 1e16, 2: -1e16, 3: 1} {
		ctx.ResultsStore.NumData[formulae.ExpandVarName("NAR_TOT", []int{key})] = []float64{value}
	}
	if got := ctx.ResultsStore.SumWhere("NAR_TOT", 0, "ASSET_CATEGORY", func(category string) bool {
		return ctx.ReadTableTxt("Assump", "Y", category, "BOND_TYPE1") == "TRAD"
	}); got != 1 {
		t.Fatalf("sum = %g, want 1 in enum order", got)
	}
}
