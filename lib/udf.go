package lib

import (
	"math"
	"strconv"
)

func MULT(integer int, mutiple int) bool {
	return integer%mutiple == 0
}

func Text[T int | float32 | float64 | string](input T) string {
	switch v := any(input).(type) {
	case int:
		return strconv.Itoa(v)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case string:
		return v
	default:
		panic("Invalid type")
	}
}
func MonthlyRate(annualRate float64) float64 {
	// Convert annual rate to monthly rate
	return math.Pow(1+annualRate/100.0, 1.0/12.0) - 1.0
}
func round4(x float64) float64 {
	return math.Round(x*10000) / 10000
}

func tIdx(startYr, startMth, curYr, curMth float64) float64 {
	return (curYr-startYr)*12 + (curMth - startMth)
}

func Idx(dims []int, idx int) int {
	if idx < 1 || idx > len(dims) {
		return 0
	}
	return dims[idx-1]
}
