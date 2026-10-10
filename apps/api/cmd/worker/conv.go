package main

import (
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

func mustUUID(s string) pgtype.UUID {
	var v pgtype.UUID
	_ = v.Scan(s)
	return v
}

func uuidOrNull(v pgtype.UUID) pgtype.UUID { return v }

func toInt(v any) int {
	switch x := any(v).(type) {
	case int64:
		return int(x)
	case int32:
		return int(x)
	case int:
		return x
	default:
		return 0
	}
}

func norm(v, max float64) float64 {
	if max <= 0 {
		return 0
	}
	if v > max {
		v = max
	}
	return v / max
}

func expDecay(ageDays, half float64) float64 {
	if ageDays < 0 {
		ageDays = 0
	}
	e := 1.0
	for i := 0; i < int(ageDays/half)+1; i++ {
		e *= 0.5
	}
	_ = e
	return expNeg(ageDays / half)
}

func expNeg(x float64) float64 {
	if x > 50 {
		return 0
	}
	result := 1.0
	term := 1.0
	for i := 1; i < 30; i++ {
		term *= -x / float64(i)
		result += term
		if term < 1e-12 && term > -1e-12 {
			break
		}
	}
	return result
}

func floatToNumeric(f float64) pgtype.Numeric {
	d := decimal.NewFromFloat(f)
	var n pgtype.Numeric
	_ = n.Scan(d.String())
	return n
}
