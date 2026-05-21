package lib

import (
	"github.com/openact/formulae"
	"github.com/openact/kit/enum"
)

var TEST_ARRAY = formulae.RegisterVectorNum(formulae.Registry, "default", "TEST_ARRAY",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		seg := coord.DimValue("ENUM_TEST_2")
		if seg == "Seg_1" {
			val = 999
		} else {
			val = 0
		}
		return
	}, "ENUM_TEST_1", "ENUM_TEST_2")

var TEST_ARRAY_TXT = formulae.RegisterVectorTxt(formulae.Registry, "default", "TEST_ARRAY_TXT",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val string) {
		seg := coord.DimValue("ENUM_TEST_2")
		if seg == "AC" {
			val = "Hello"
		} else {
			val = "World"
		}
		return
	}, "ENUM_TEST_1", "ENUM_TEST_2")

var TEST_ARRAY_NEW = formulae.RegisterVectorNum(formulae.Registry, "default", "TEST_ARRAY_NEW",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		class := coord.DimValue("ASSET_CLASS")

		if class == "IG_Tradable" {
			val = 777
		} else {
			val = 0
		}
		return
	}, "ASSET_CLASS")

var ASSET_MIX = formulae.RegisterVectorNum(formulae.Registry, "default", "ASSET_MIX",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		sim := ctx.Sim
		class := coord.DimValue("ASSET_CLASS")

		val = ctx.ReadTableNum("SAA_Sims", "Y", sim, class)
		return
	}, "ASSET_CLASS")

var TRAD_BOND_DIST = formulae.RegisterVectorNum(formulae.Registry, "default", "TRAD_BOND_DIST",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		ratingBand := coord.DimValue("ENUM_BOND_RATING")
		termBand := coord.DimValue("ENUM_BOND_TERM")

		val = ctx.ReadTableNum("AssetTradBondDistByRatingByTerm", "Y", ratingBand, termBand)
		return
	}, "ENUM_BOND_RATING", "ENUM_BOND_TERM")

var ALT_BOND_DIST = formulae.RegisterVectorNum(formulae.Registry, "default", "ALT_BOND_DIST",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		ratingBand := coord.DimValue("ENUM_BOND_RATING")
		termBand := coord.DimValue("ENUM_BOND_TERM")

		val = ctx.ReadTableNum("AssetAltBondDistByRatingByTerm", "Y", ratingBand, termBand)
		return
	}, "ENUM_BOND_RATING", "ENUM_BOND_TERM")

var TRAD_BOND_DUR = formulae.RegisterVectorNum(formulae.Registry, "default", "TRAD_BOND_DUR",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		termBand := coord.DimValue("ENUM_BOND_TERM")

		val = ctx.ReadTableNum("AssetTradBondAssump", "Y", termBand, "DURATION")
		return
	}, "ENUM_BOND_TERM")

var ALT_BOND_DUR = formulae.RegisterVectorNum(formulae.Registry, "default", "ALT_BOND_DUR",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		termBand := coord.DimValue("ENUM_BOND_TERM")

		val = ctx.ReadTableNum("AssetAltBondAssump", "Y", termBand, "DURATION")
		return
	}, "ENUM_BOND_TERM")

var TRAD_BOND_MIX_BY_TERM = formulae.RegisterVectorNum(formulae.Registry, "default", "TRAD_BOND_MIX_BY_TERM",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		termBand := coord.DimValue("ENUM_BOND_TERM")

		val = formulae.ArraySumIf(ctx, i, TRAD_BOND_DIST, "ENUM_BOND_TERM", termBand)

		return
	}, "ENUM_BOND_TERM")

var ALT_BOND_MIX_BY_TERM = formulae.RegisterVectorNum(formulae.Registry, "default", "ALT_BOND_MIX_BY_TERM",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		termBand := coord.DimValue("ENUM_BOND_TERM")

		val = formulae.ArraySumIf(ctx, i, ALT_BOND_DIST, "ENUM_BOND_TERM", termBand)
		return
	}, "ENUM_BOND_TERM")

var STRESS_BOND_SPD_PARAMS = formulae.RegisterVectorNum(formulae.Registry, "default", "STRESS_BOND_SPD_PARAMS",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		ratingBand := coord.DimValue("ENUM_BOND_RATING")
		termBand := coord.DimValue("ENUM_BOND_TERM")

		val = ctx.ReadTableNum("StressBondSpdByRatingByTerm", "Y", ratingBand, termBand)
		return
	}, "ENUM_BOND_RATING", "ENUM_BOND_TERM")

var ASSET_RETURN_RATE = formulae.RegisterVectorNum(formulae.Registry, "default", "ASSET_RETURN_RATE",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		class := coord.DimValue("ASSET_CLASS")

		val = ctx.ReadTableNum("Assump", "Y", class, "RET_RATE")
		return
	}, "ASSET_CLASS")

var ASSET_SD_RATE = formulae.RegisterVectorNum(formulae.Registry, "default", "ASSET_SD_RATE",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		class := coord.DimValue("ASSET_CLASS")

		val = ctx.ReadTableNum("Assump", "Y", class, "SD_RATE")
		return
	}, "ASSET_CLASS")

var ASSET_HKD_MIX = formulae.RegisterVectorNum(formulae.Registry, "default", "ASSET_HKD_MIX",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		class := coord.DimValue("ASSET_CLASS")

		val = ctx.ReadTableNum("Assump", "Y", class, "ASSET_HKD_MIX")
		return
	}, "ASSET_CLASS")

var ASSET_USD_MIX = formulae.RegisterVectorNum(formulae.Registry, "default", "ASSET_USD_MIX",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		class := coord.DimValue("ASSET_CLASS")

		val = ctx.ReadTableNum("Assump", "Y", class, "ASSET_USD_MIX")
		return
	}, "ASSET_CLASS")

var ASSET_BAL = formulae.RegisterVectorNum(formulae.Registry, "default", "ASSET_BAL",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		totalAsset := ctx.Mp.Num("TOT_ASSET")
		rs := ctx.ResultsStore
		val = totalAsset * rs.GetArrayNum("ASSET_MIX", coord, i)
		return
	}, "ASSET_CLASS")

var NAR_TOT = formulae.RegisterVectorNum(formulae.Registry, "default", "NAR_TOT",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		rs := ctx.ResultsStore
		val = rs.GetArrayNum("ASSET_BAL", coord, i) * ctx.ReadTableNum("Assump", "Y", coord.DimValue("ASSET_CLASS"), "NAR_TOT")
		return
	}, "ASSET_CLASS")

var NAR_FI = formulae.RegisterVectorNum(formulae.Registry, "default", "NAR_FI",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		rs := ctx.ResultsStore
		val = rs.GetArrayNum("ASSET_BAL", coord, i) * ctx.ReadTableNum("Assump", "Y", coord.DimValue("ASSET_CLASS"), "NAR_FI")
		return
	}, "ASSET_CLASS")

var NAR_FI_HKD = formulae.RegisterVectorNum(formulae.Registry, "default", "NAR_FI_HKD",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		rs := ctx.ResultsStore
		val = rs.GetArrayNum("NAR_FI", coord, i) * rs.GetArrayNum("ASSET_HKD_MIX", coord, i)
		return
	}, "ASSET_CLASS")

var NAR_FI_USD = formulae.RegisterVectorNum(formulae.Registry, "default", "NAR_FI_USD",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		rs := ctx.ResultsStore
		val = rs.GetArrayNum("NAR_FI", coord, i) * rs.GetArrayNum("ASSET_USD_MIX", coord, i)
		return
	}, "ASSET_CLASS")

var NAR_EQ = formulae.RegisterVectorNum(formulae.Registry, "default", "NAR_EQ",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		rs := ctx.ResultsStore
		val = rs.GetArrayNum("ASSET_BAL", coord, i) * ctx.ReadTableNum("Assump", "Y", coord.DimValue("ASSET_CLASS"), "NAR_EQ")
		return
	}, "ASSET_CLASS")

var NAR_PROP = formulae.RegisterVectorNum(formulae.Registry, "default", "NAR_PROP",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		rs := ctx.ResultsStore
		val = rs.GetArrayNum("ASSET_BAL", coord, i) * ctx.ReadTableNum("Assump", "Y", coord.DimValue("ASSET_CLASS"), "NAR_PROP")
		return
	}, "ASSET_CLASS")

var TOT_NAR = formulae.RegisterVectorNum(formulae.Registry, "default", "TOT_NAR",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		rs := ctx.ResultsStore
		val = rs.GetArrayNum("NAR_FI", coord, i) + rs.GetArrayNum("NAR_EQ", coord, i) + rs.GetArrayNum("NAR_PROP", coord, i)
		return
	}, "ASSET_CLASS")

var RC_ASSET_EQ = formulae.RegisterVectorNum(formulae.Registry, "default", "RC_ASSET_EQ",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		class := coord.DimValue("ASSET_CLASS")
		rs := ctx.ResultsStore
		val = rs.GetArrayNum("NAR_EQ", coord, i) * ctx.ReadTableNum("Assump", "Y", class, "RC_FAC_EQ")
		return
	}, "ASSET_CLASS")

var RC_ASSET_PROP = formulae.RegisterVectorNum(formulae.Registry, "default", "RC_ASSET_PROP",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		class := coord.DimValue("ASSET_CLASS")
		rs := ctx.ResultsStore
		val = rs.GetArrayNum("NAR_PROP", coord, i) * ctx.ReadTableNum("Assump", "Y", class, "RC_FAC_PROP")
		return
	}, "ASSET_CLASS")

var NAR_TOT_HKD = formulae.RegisterVectorNum(formulae.Registry, "default", "NAR_TOT_HKD",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		rs := ctx.ResultsStore
		val = rs.GetArrayNum("NAR_TOT", coord, i) * rs.GetArrayNum("ASSET_HKD_MIX", coord, i)
		return
	}, "ASSET_CLASS")

var NAR_TOT_USD = formulae.RegisterVectorNum(formulae.Registry, "default", "NAR_TOT_USD",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		rs := ctx.ResultsStore
		val = rs.GetArrayNum("NAR_TOT", coord, i) * rs.GetArrayNum("ASSET_USD_MIX", coord, i)
		return
	}, "ASSET_CLASS")

var PIVOT_RC_SPD_BOND_TRAD = formulae.RegisterVectorNum(formulae.Registry, "default", "PIVOT_RC_SPD_BOND_TRAD",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		rs := ctx.ResultsStore
		spdParams := rs.GetArrayNum("STRESS_BOND_SPD_PARAMS", coord, i)
		tradBondDist := rs.GetArrayNum("TRAD_BOND_DIST", coord, i)
		bondTrad := rs.GetNum("BOND_TRAD", i)

		val = bondTrad * tradBondDist * spdParams
		return
	}, "ENUM_BOND_RATING", "ENUM_BOND_TERM")

var PIVOT_RC_SPD_BOND_ALT = formulae.RegisterVectorNum(formulae.Registry, "default", "PIVOT_RC_SPD_BOND_ALT",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		rs := ctx.ResultsStore
		spdParams := rs.GetArrayNum("STRESS_BOND_SPD_PARAMS", coord, i)
		altBondDist := rs.GetArrayNum("ALT_BOND_DIST", coord, i)
		bondAlt := rs.GetNum("BOND_ALT", i)

		val = bondAlt * altBondDist * spdParams
		return
	}, "ENUM_BOND_RATING", "ENUM_BOND_TERM")

var PIVOT_RC_SPD_BOND = formulae.RegisterVectorNum(formulae.Registry, "default", "PIVOT_RC_SPD_BOND",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		rs := ctx.ResultsStore
		val = rs.GetArrayNum("PIVOT_RC_SPD_BOND_TRAD", coord, i) + rs.GetArrayNum("PIVOT_RC_SPD_BOND_ALT", coord, i)
		return
	}, "ENUM_BOND_RATING", "ENUM_BOND_TERM")

var RF_ECON_SCENARIO = formulae.RegisterVectorNum(formulae.Registry, "default", "RF_ECON_SCENARIO",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		econ := coord.DimValue("ENUM_ECON")
		scenario := coord.DimValue("ENUM_SCENARIOS_INT")
		rs := ctx.ResultsStore
		term := rs.GetNum("BOND_DUR_INT", i)
		val = ctx.ReadTableNum("RF_Curves", "Y", econ, scenario, formulae.Text(term))
		return
	}, "ENUM_ECON", "ENUM_SCENARIOS_INT")

var PCR_MKT_RISK = formulae.RegisterVectorNum(formulae.Registry, "default", "PCR_MKT_RISK",
	func(ctx *formulae.ProjContext, i int, coord *enum.Coordinate) (val float64) {
		mktRisk := coord.DimValue("ENUM_MKT_RISK")
		rs := ctx.ResultsStore
		switch mktRisk {
		case "INT":
			val = rs.GetNum("PCR_INT", i)
		case "SPD":
			val = rs.GetNum("PCR_SPD", i)
		case "EQ":
			val = rs.GetNum("PCR_EQ", i)
		case "PROP":
			val = rs.GetNum("PCR_PROP", i)
		case "FX":
			val = rs.GetNum("PCR_FX", i)
		default:
			val = 0.0
		}
		return
	}, "ENUM_MKT_RISK")
