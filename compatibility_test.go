package main

import (
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/openact/formulae"
	"github.com/openact/kit/cache/v3"
	"github.com/openact/kit/cli"
	"github.com/openact/kit/resultmeta"
	"gopkg.in/yaml.v3"
)

// Legacy engine 0fed87b with the checked-in SAA inputs, at projection step zero.
var saaLegacyKRI = map[string]map[string]float64{
	"1": {"PORT_RETURN_RATE": 0.072596508817349775, "PORT_RISK": 0.078562425700346744, "PCR_TOTAL": 643.35315928676221},
	"2": {"PORT_RETURN_RATE": 0.038686490409458672, "PORT_RISK": 0.034890086531584895, "PCR_TOTAL": 527.93935544470241},
	"3": {"PORT_RETURN_RATE": 0.040546490409458666, "PORT_RISK": 0.025127929131854094, "PCR_TOTAL": 525.70255088440956},
	"4": {"PORT_RETURN_RATE": 0.032064550071604171, "PORT_RISK": 0.010747278005792922, "PCR_TOTAL": 479.60934099643197},
}

func localSAASettings(t *testing.T) formulae.Settings {
	t.Helper()
	app, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	output := t.TempDir()
	settings, err := formulae.BuildRunSettings(&cli.ResolvedPaths{
		ConfigDir: filepath.Join(app, "config"), ConfigFile: filepath.Join(app, "config", "config.yaml"),
		InputDir: filepath.Join(app, "input"), InputFile: filepath.Join(app, "input", "config.yaml"),
		OutputDir: output, WorkDir: filepath.Join(output, ".work"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(settings) == 0 {
		t.Fatal("no SAA runs configured")
	}
	return settings
}

func localSAAStructure(t *testing.T, setting *formulae.Setting) *formulae.Structure {
	t.Helper()
	structure, err := formulae.NewStructure(
		filepath.Join(setting.ConfPaths.StructureDir, setting.Structure+".csv"),
		formulae.Registry, setting.ConfPaths.ProductDir, setting.ConfPaths.AccumulationDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := structure.ValidateSetting(setting); err != nil {
		t.Fatal(err)
	}
	return structure
}

func assertSAANumber(t *testing.T, got, want float64) {
	t.Helper()
	if math.IsNaN(got) || math.IsInf(got, 0) || math.Abs(got-want) > 1e-8+1e-10*math.Abs(want) {
		t.Fatalf("got %.17g, want %.17g", got, want)
	}
}

func TestSAAConfiguredRunsPreserveFinancialResults(t *testing.T) {
	for name, setting := range localSAASettings(t) {
		t.Run(name, func(t *testing.T) {
			if !reflect.DeepEqual(setting.GroupBy, []string{"SP_CODE", "ACCUM"}) ||
				!reflect.DeepEqual(setting.Statistics, []formulae.StatisticsSpec{{Component: "ALM", Views: []string{"SP_CODE"}}}) {
				t.Fatalf("unexpected grouping/statistics: %v, %v", setting.GroupBy, setting.Statistics)
			}
			structure := localSAAStructure(t, setting)
			if !reflect.DeepEqual(structure.Names, []string{"ALM"}) {
				t.Fatalf("unexpected components: %v", structure.Names)
			}
			if !reflect.DeepEqual(setting.Simulations, []string{"1", "2", "3", "4"}) {
				t.Fatalf("unexpected configured scenarios: %v", setting.Simulations)
			}
			setting.SeriatimResults = false
			for _, sim := range setting.Simulations {
				t.Run(sim, func(t *testing.T) {
					context := formulae.NewScenarioContext(setting, sim)
					if err := structure.Proj(context); err != nil {
						t.Fatal(err)
					}
					bySP := context.GetResults("ALM", "SP_CODE")
					byAccum := context.GetResults("ALM", "ACCUM")
					if len(bySP) != 1 || len(byAccum) != 1 || bySP["1"] == nil || byAccum["FUND"] == nil {
						t.Fatalf("unexpected product groups: %v, %v", bySP, byAccum)
					}
					if !reflect.DeepEqual(bySP["1"].NumData, byAccum["FUND"].NumData) {
						t.Fatal("independent SP_CODE and ACCUM views disagree")
					}
					if len(bySP["1"].NumData) != len(saaLegacyKRI[sim]) {
						t.Fatalf("unexpected aggregated variables: %v", bySP["1"].NumData)
					}
					for variable, want := range saaLegacyKRI[sim] {
						assertSAANumber(t, bySP["1"].GetNum(variable, 0), want)
					}
				})
			}
		})
	}
}

func TestSAAScenarioStatisticsAndOutput(t *testing.T) {
	for name, setting := range localSAASettings(t) {
		t.Run(name, func(t *testing.T) {
			if err := formulae.RunProjection(&formulae.ProjContext{Setting: setting}, setting, localSAAStructure(t, setting)); err != nil {
				t.Fatal(err)
			}
			body, err := os.ReadFile(filepath.Join(setting.ResultPath, "metadata", "metadata.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			var metadata resultmeta.File
			if err := yaml.Unmarshal(body, &metadata); err != nil {
				t.Fatal(err)
			}
			wantFiles := map[string]bool{
				"ALM_by_SP_CODE.fac": false, "ALM_by_ACCUM.fac": false,
				"ALM_by_SP_CODE_average.fac":   false,
				"stats/ALM_by_SP_CODE_min.fac": false, "stats/ALM_by_SP_CODE_max.fac": false,
			}
			for _, item := range metadata.Catalog.Products {
				if _, ok := wantFiles[item.FileName]; !ok || item.Component != "ALM" {
					t.Fatalf("unexpected catalog entry: %+v", item)
				}
				wantFiles[item.FileName] = true
				table, err := cache.LoadTable(filepath.Join(setting.ResultPath, filepath.FromSlash(item.FileName)),
					cache.LoadOptions{SuppressLoadLog: true})
				if err != nil {
					t.Fatal(err)
				}
				if item.Statistic == "" {
					for sim, variables := range saaLegacyKRI {
						for variable, want := range variables {
							got, found, err := table.Float([]string{sim, "ALM", item.Group, variable}, "202412")
							if err != nil || !found {
								t.Fatalf("missing scenario value %s/%s: %v", sim, variable, err)
							}
							assertSAANumber(t, got, want)
						}
					}
					continue
				}
				if item.View != "SP_CODE" || item.Group != "1" {
					t.Fatalf("unexpected statistics identity: %+v", item)
				}
				for _, variable := range []string{"PORT_RETURN_RATE", "PORT_RISK", "PCR_TOTAL"} {
					var values []float64
					average := 0.0
					for _, sim := range setting.Simulations {
						value := saaLegacyKRI[sim][variable]
						values = append(values, value)
						average += value / float64(len(setting.Simulations))
					}
					want, ok := map[string]float64{"average": average, "min": slices.Min(values), "max": slices.Max(values)}[item.Statistic]
					if !ok {
						t.Fatalf("unexpected statistic: %s", item.Statistic)
					}
					got, found, err := table.Float([]string{"ALM", "1", variable}, "202412")
					if err != nil || !found {
						t.Fatalf("missing statistic %s/%s: %v", item.Statistic, variable, err)
					}
					assertSAANumber(t, got, want)
				}
			}
			for file, found := range wantFiles {
				if !found {
					t.Errorf("missing catalog file: %s", file)
				}
			}
			if _, err := os.Stat(filepath.Join(setting.ResultPath, "metadata", "completion.yaml")); err != nil {
				t.Fatal(err)
			}
			for _, sim := range setting.Simulations {
				path := filepath.Join(setting.ResultPath, "seriatim", "ALM-1-FUND01-sp1-sim"+sim+".fac")
				table, err := cache.LoadTable(path, cache.LoadOptions{SuppressLoadLog: true})
				if err != nil {
					t.Fatal(err)
				}
				for variable, want := range saaLegacyKRI[sim] {
					got, found, err := table.Float([]string{sim, "ALM", "1", "FUND01", "1", variable}, "202412")
					if err != nil || !found {
						t.Fatalf("missing seriatim value %s/%s: %v", sim, variable, err)
					}
					assertSAANumber(t, got, want)
				}
				variable := formulae.ExpandVarName("ASSET_MIX", []int{1})
				got, found, err := table.Float([]string{sim, "ALM", "1", "FUND01", "1", variable}, "202412")
				if err != nil || !found {
					t.Fatalf("missing seriatim array %s/%s: %v", sim, variable, err)
				}
				ctx := &formulae.ProjContext{Setting: setting}
				assertSAANumber(t, got, ctx.ReadTableNum("SAA_Sims", "Y", sim, ctx.EnumTxt("ASSET_CLASS", 1)))
			}
		})
	}
}
