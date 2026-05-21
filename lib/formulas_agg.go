package lib

import (
	"math"

	"github.com/openact/formulae"
)

// Fund Level
var TEST_STR_VAR = formulae.RegisterScalarTxt(formulae.Registry, "default", "TEST_STR_VAR", func(ctx *formulae.ProjContext, i int, dims ...int) (val string) {
	val = "Hello, World!"
	return
})

var PORT_RETURN_RATE = formulae.RegisterScalarNum(formulae.Registry, "default", "PORT_RETURN_RATE", func(ctx *formulae.ProjContext, i int, dims ...int) (val float64) {
	val = formulae.SumProduct(ctx, i, ASSET_MIX, ASSET_RETURN_RATE)
	return
})

var PORT_RISK = formulae.RegisterScalarNum(formulae.Registry, "default", "PORT_RISK",
	func(ctx *formulae.ProjContext, i int, dims ...int) (val float64) {
		val = formulae.ArrayAggregateByCorr(ctx, i, "ASSET_CORR_MATRIX", ASSET_MIX)
		return
	})

var PORT_NAR_HKD = formulae.RegisterScalarNum(formulae.Registry, "default", "PORT_NAR_HKD", func(ctx *formulae.ProjContext, i int, dims ...int) (val float64) {
	val = formulae.ArraySum(ctx, i, NAR_TOT_HKD)
	return
})

var PORT_NAR_USD = formulae.RegisterScalarNum(formulae.Registry, "default", "PORT_NAR_USD", func(ctx *formulae.ProjContext, i int, dims ...int) (val float64) {
	val = formulae.ArraySum(ctx, i, NAR_TOT_USD)
	return
})

var PCR_FX = formulae.RegisterScalarNum(formulae.Registry, "default", "PCR_FX", func(ctx *formulae.ProjContext, i int, dims ...int) (val float64) {
	rs := ctx.ResultsStore
	val = rs.GetNum("PORT_NAR_HKD", i)*rs.GetNum("RC_FAC_HKD", i) + rs.GetNum("PORT_NAR_USD", i)*rs.GetNum("RC_FAC_USD", i)
	return
})

var PCR_EQ = formulae.RegisterScalarNum(formulae.Registry, "default", "PCR_EQ", func(ctx *formulae.ProjContext, i int, dims ...int) (val float64) {
	val = formulae.ArraySum(ctx, i, RC_ASSET_EQ)
	return
})

var PCR_PROP = formulae.RegisterScalarNum(formulae.Registry, "default", "PCR_PROP", func(ctx *formulae.ProjContext, i int, dims ...int) (val float64) {
	val = formulae.ArraySum(ctx, i, RC_ASSET_PROP)
	return
})

var PCR_SPD = formulae.RegisterScalarNum(formulae.Registry, "default", "PCR_SPD", func(ctx *formulae.ProjContext, i int, dims ...int) (val float64) {
	val = -formulae.ArraySum(ctx, i, PIVOT_RC_SPD_BOND)
	return
})

var BOND_TRAD = formulae.RegisterScalarNum(formulae.Registry, "default", "BOND_TRAD", func(ctx *formulae.ProjContext, i int, dims ...int) (val float64) {
	//AC,IG_Tradable,HY,Securitized_CMBS
	val = formulae.ArraySumIf(ctx, i, NAR_FI, "ASSET_CLASS", "AC") +
		formulae.ArraySumIf(ctx, i, NAR_FI, "ASSET_CLASS", "IG_Tradable") +
		formulae.ArraySumIf(ctx, i, NAR_FI, "ASSET_CLASS", "HY") +
		formulae.ArraySumIf(ctx, i, NAR_FI, "ASSET_CLASS", "Securitized_CMBS")
	return
})

var BOND_ALT = formulae.RegisterScalarNum(formulae.Registry, "default", "BOND_ALT", func(ctx *formulae.ProjContext, i int, dims ...int) (val float64) {
	//Loan_Private_Credit
	val = formulae.ArraySumIf(ctx, i, NAR_FI, "ASSET_CLASS", "Loan_Private_Credit")
	return
})

var BOND_DUR = formulae.RegisterScalarNum(formulae.Registry, "default", "BOND_DUR", func(ctx *formulae.ProjContext, i int, dims ...int) (val float64) {
	bondTrad := ctx.ResultsStore.GetNum("BOND_TRAD", i)
	bondAlt := ctx.ResultsStore.GetNum("BOND_ALT", i)
	if bondTrad+bondAlt == 0 {
		val = 0
	} else {
		durBondTrad := formulae.SumProduct(ctx, i, TRAD_BOND_MIX_BY_TERM, TRAD_BOND_DUR)
		durBondAlt := formulae.SumProduct(ctx, i, ALT_BOND_MIX_BY_TERM, ALT_BOND_DUR)
		val = (bondTrad*durBondTrad + bondAlt*durBondAlt) / (bondTrad + bondAlt)
	}
	return
})

var BOND_DUR_INT = formulae.RegisterScalarNum(formulae.Registry, "default", "BOND_DUR_INT", func(ctx *formulae.ProjContext, i int, dims ...int) (val float64) {
	rs := ctx.ResultsStore
	val = rs.GetNum("BOND_DUR", i)
	val = math.RoundToEven(val)
	return
})

var SUM_NAR_FI_HKD = formulae.RegisterScalarNum(formulae.Registry, "default", "SUM_NAR_FI_HKD", func(ctx *formulae.ProjContext, i int, dims ...int) (val float64) {
	val = formulae.ArraySum(ctx, i, NAR_FI_HKD)
	return
})

var SUM_NAR_FI_USD = formulae.RegisterScalarNum(formulae.Registry, "default", "SUM_NAR_FI_USD", func(ctx *formulae.ProjContext, i int, dims ...int) (val float64) {
	val = formulae.ArraySum(ctx, i, NAR_FI_USD)
	return
})

var HKD_BOND_MV_CHG = formulae.RegisterScalarNum(formulae.Registry, "default", "HKD_BOND_MV_CHG", func(ctx *formulae.ProjContext, i int, dims ...int) (val float64) {
	intBitingScen := ctx.Mp.Txt("INT_BITING")
	rs := ctx.ResultsStore

	bondDur := rs.GetNum("BOND_DUR", i)
	term := rs.GetNum("BOND_DUR_INT", i)
	val = (ctx.ReadTableNum("RF_Curves", "Y", "HKD", intBitingScen, formulae.Text(term)) -
		ctx.ReadTableNum("RF_Curves", "Y", "HKD", "BASE", formulae.Text(term))) * bondDur * rs.GetNum("SUM_NAR_FI_HKD", i)
	return
})

var USD_BOND_MV_CHG = formulae.RegisterScalarNum(formulae.Registry, "default", "USD_BOND_MV_CHG", func(ctx *formulae.ProjContext, i int, dims ...int) (val float64) {
	intBitingScen := ctx.Mp.Txt("INT_BITING")
	rs := ctx.ResultsStore

	bondDur := rs.GetNum("BOND_DUR", i)
	term := rs.GetNum("BOND_DUR_INT", i)
	val = (ctx.ReadTableNum("RF_Curves", "Y", "USD", intBitingScen, formulae.Text(term)) -
		ctx.ReadTableNum("RF_Curves", "Y", "USD", "BASE", formulae.Text(term))) * (-bondDur) * rs.GetNum("SUM_NAR_FI_USD", i)
	return
})

var PORT_BOND_MV = formulae.RegisterScalarNum(formulae.Registry, "default", "PORT_BOND_MV", func(ctx *formulae.ProjContext, i int, dims ...int) (val float64) {
	rs := ctx.ResultsStore
	val = rs.GetNum("SUM_NAR_FI_HKD", i) + rs.GetNum("SUM_NAR_FI_USD", i)
	return
})

var INT_RISK_SCAL = formulae.RegisterScalarNum(formulae.Registry, "default", "INT_RISK_SCAL", func(ctx *formulae.ProjContext, i int, dims ...int) (val float64) {
	mp := ctx.Mp
	rs := ctx.ResultsStore
	portBondMV := rs.GetNum("PORT_BOND_MV", i)
	if portBondMV == 0 || rs.GetNum("BOND_DUR", i) == 0 {
		val = 0
	} else {
		val = (mp.Num("TOT_LIAB")*mp.Num("LIAB_DUR_PCR"))/(portBondMV*rs.GetNum("BOND_DUR", i)) - 1
	}
	return
})

var PORT_BOND_MV_CHG = formulae.RegisterScalarNum(formulae.Registry, "default", "PORT_BOND_MV_CHG", func(ctx *formulae.ProjContext, i int, dims ...int) (val float64) {
	rs := ctx.ResultsStore
	val = rs.GetNum("HKD_BOND_MV_CHG", i) + rs.GetNum("USD_BOND_MV_CHG", i)
	return
})

var PCR_INT = formulae.RegisterScalarNum(formulae.Registry, "default", "PCR_INT", func(ctx *formulae.ProjContext, i int, dims ...int) (val float64) {
	rs := ctx.ResultsStore
	val = rs.GetNum("PORT_BOND_MV_CHG", i) * rs.GetNum("INT_RISK_SCAL", i)
	return
})

var PCR_TOTAL = formulae.RegisterScalarNum(formulae.Registry, "default", "PCR_TOTAL", func(ctx *formulae.ProjContext, i int, dims ...int) (val float64) {
	intBitingScen := ctx.Mp.Txt("INT_BITING")
	if intBitingScen == "INT_UP" {
		val = formulae.ArrayAggregateByCorr(ctx, i, "MKT_RISK_CORR_INTUP", PCR_MKT_RISK)
	} else if intBitingScen == "INT_DN" {
		val = formulae.ArrayAggregateByCorr(ctx, i, "MKT_RISK_CORR_INTDN", PCR_MKT_RISK)
	}
	return
})
