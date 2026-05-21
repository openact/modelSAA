package lib

import "github.com/openact/formulae"

var CALENDAR_YR = formulae.RegisterScalarNum(formulae.Registry, "default", "CALENDAR_YR", func(ctx *formulae.ProjContext, i int, dims ...int) (val float64) {
	s := ctx.Setting
	//startYear := int(s.START_YEAR)
	//startMonth := int(s.START_MONTH)
	startYear := s.GetParameter("startYear")
	startMonth := s.GetParameter("startMonth")

	// Calculate the total months and convert to years
	totalMonths := startMonth - 1 + i
	year := startYear + totalMonths/12

	val = float64(year)
	return
})

var CALENDAR_MTH = formulae.RegisterScalarNum(formulae.Registry, "default", "CALENDAR_MTH", func(ctx *formulae.ProjContext, i int, dims ...int) (val float64) {
	s := ctx.Setting
	//startMth := int(s.START_MONTH)
	startMth := s.GetParameter("startMonth")

	currentMth := (startMth-1+i)%12 + 1
	val = float64(currentMth)
	return
})

var CALENDAR_DATE = formulae.RegisterScalarNum(formulae.Registry, "default", "CALENDAR_DATE", func(ctx *formulae.ProjContext, i int, dims ...int) (val float64) {
	rs := ctx.ResultsStore
	val = rs.GetNum("CALENDAR_YR", i)*100 + rs.GetNum("CALENDAR_MTH", i)
	return
})
