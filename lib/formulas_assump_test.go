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
	table, err := cache.LoadTable(path, cache.LoadOptions{PreindexRows: true, SuppressLoadLog: true})
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
	return formulae.NewProjContext(&formulae.Setting{
		Global: &formulae.Global{},
		Enums:  map[string]*enum.Enum{"ASSET_CATEGORY": categories},
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

func TestAssumpReadsLiteralClassValues(t *testing.T) {
	for _, value := range []string{"Bond", "bOnD", "", "Unknown_Class"} {
		t.Run(value, func(t *testing.T) {
			ctx := assetMappingContext(t, "!2,ASSET_CATEGORY,ASSET_CLASS\n*,IG_Tradable,"+value+"\n")
			if got := ctx.ReadTableTxt("Assump", "Y", "IG_Tradable", "ASSET_CLASS"); got != value {
				t.Fatalf("class = %q, want literal value %q", got, value)
			}
			if got := ctx.ReadTableTxt("Assump", "Y", "ig_tradable", "ASSET_CLASS"); got != value {
				t.Fatalf("case-insensitive row lookup = %q, want %q", got, value)
			}
			delete(ctx.Setting.Tables, "assump")
			expectAssetMappingPanic(t, "not found", func() {
				ctx.ReadTableTxt("Assump", "Y", "IG_Tradable", "ASSET_CLASS")
			})
		})
	}
}

func TestAssumpReadsRejectMissingCells(t *testing.T) {
	t.Run("missing category", func(t *testing.T) {
		ctx := assetMappingContext(t, "!2,ASSET_CATEGORY,ASSET_CLASS\n*,IG_Tradable,Bond\n")
		expectAssetMappingPanic(t, "cell not found", func() {
			ctx.ReadTableTxt("Assump", "Y", "Cash", "ASSET_CLASS")
		})
	})
	t.Run("missing class column", func(t *testing.T) {
		ctx := assetMappingContext(t, "!2,ASSET_CATEGORY,BOND_TYPE1\n*,IG_Tradable,TRAD\n")
		expectAssetMappingPanic(t, "not found", func() {
			ctx.ReadTableTxt("Assump", "Y", "IG_Tradable", "ASSET_CLASS")
		})
	})
	t.Run("missing bond type column", func(t *testing.T) {
		ctx := assetMappingContext(t, "!2,ASSET_CATEGORY,ASSET_CLASS\n*,IG_Tradable,Bond\n*,Cash,Cash\n")
		if err := NAR_FI.InitializeWithEnums(ctx.Setting); err != nil {
			t.Fatal(err)
		}
		if err := ctx.ResultsStore.BindDefinitions(NAR_FI); err != nil {
			t.Fatal(err)
		}
		expectAssetMappingPanic(t, "not found", func() {
			BOND_TRAD.NumFunc(ctx, 0, nil)
		})
	})
}

func TestAssetMixDoesNotReadAssump(t *testing.T) {
	ctx := assetMappingContext(t, "!2,ASSET_CATEGORY,ASSET_CLASS\n*,IG_Tradable,Bond\n")
	path := filepath.Join(t.TempDir(), "SAA_Sims.fac")
	if err := os.WriteFile(path, []byte("!2,SIMULATION,IG_Tradable,Cash\n*,1,0.25,0.75\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	table, err := cache.LoadTable(path, cache.LoadOptions{SuppressLoadLog: true})
	if err != nil {
		t.Fatal(err)
	}
	ctx.Setting.Tables["saa_sims"] = table
	delete(ctx.Setting.Tables, "assump")
	if err := ASSET_MIX.InitializeWithEnums(ctx.Setting); err != nil {
		t.Fatal(err)
	}
	for i, want := range []float64{0.25, 0.75} {
		if got := ASSET_MIX.NumFunc(ctx, 0, []int{i + 1}); got != want {
			t.Fatalf("ASSET_MIX(%d) = %g, want %g", i+1, got, want)
		}
	}
}

func TestAssumpTableRejectsDuplicateCategory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Assump.fac")
	content := "!2,ASSET_CATEGORY,ASSET_CLASS,BOND_TYPE1\n*,IG_Tradable,Bond,TRAD\n*,ig_tradable,Equity,NONE\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.LoadTable(path, cache.LoadOptions{PreindexRows: true, SuppressLoadLog: true}); err == nil ||
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
		{"literal group comparison", "trad", "none", 0, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			content := fmt.Sprintf("!2,ASSET_CATEGORY,ASSET_CLASS,BOND_TYPE1\n*,Cash,Cash,%s\n*,IG_Tradable,Bond,%s\n",
				tt.cashType1, tt.tradableType1)
			ctx := assetMappingContext(t, content)
			if err := NAR_FI.InitializeWithEnums(ctx.Setting); err != nil {
				t.Fatal(err)
			}
			if err := ctx.ResultsStore.BindDefinitions(NAR_FI); err != nil {
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
