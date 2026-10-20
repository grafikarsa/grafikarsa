package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
	"grafikarsa/api/internal/repo"
)

func shouldSkipNotif(ctx context.Context, q *repo.Queries, userID, topic string) bool {
	if userID == "" {
		return false
	}
	pref, err := q.GetNotifPrefs(ctx, mustUUID(userID))
	if err != nil {
		return false
	}
	switch topic {
	case "notify.portfolio_liked":
		return !pref.PortfolioLiked
	case "notify.new_comment", "notify.reply_comment":
		return !pref.NewComment
	case "notify.new_follower":
		return !pref.NewFollower
	}
	return false
}

func mustUUID(s string) pgtype.UUID {
	var v pgtype.UUID
	_ = v.Scan(s)
	return v
}

func uuidOrNull(v pgtype.UUID) pgtype.UUID { return v }

func toInt(v any) int {
	switch x := v.(type) {
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

func uuidStr(v pgtype.UUID) string {
	if !v.Valid {
		return ""
	}
	b := v.Bytes
	const hex = "0123456789abcdef"
	out := make([]byte, 36)
	j := 0
	for i, c := range b {
		if i == 4 || i == 6 || i == 8 || i == 10 {
			out[j] = '-'
			j++
		}
		out[j] = hex[c>>4]
		out[j+1] = hex[c&0xf]
		j += 2
	}
	return string(out)
}

// toPgDate mengonversi "YYYY-MM-DD" menjadi pgtype.Date.
func toPgDate(day string) pgtype.Date {
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: t, Valid: true}
}
