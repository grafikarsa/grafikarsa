package http

import (
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
	"grafikarsa/api/internal/repo"
)

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

func strOrEmpty(v pgtype.Text) string {
	if !v.Valid {
		return ""
	}
	return v.String
}

func toPgIP(s string) *netip.Addr {
	addr, err := netip.ParseAddr(s)
	if err != nil {
		return nil
	}
	return &addr
}

func ipStr(v *netip.Addr) string {
	if v == nil {
		return ""
	}
	return v.String()
}

func toPgTimestamptzTime(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func updatePwParams(id pgtype.UUID, hash string) repo.UpdatePasswordHashParams {
	return repo.UpdatePasswordHashParams{ID: id, PasswordHash: hash}
}

func checkBcrypt(pw, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// publicUser: bentuk user yang konsisten dengan GET /me (id = UUID),
// supaya FE tidak salah memakai username sebagai id.
func publicUser(u repo.GetUserByUsernameRow) map[string]any {
	return map[string]any{
		"id": uuidStr(u.ID), "username": u.Username,
		"name": u.Name, "email": u.Email,
		"role_primary": u.RolePrimary, "status": u.Status,
		"must_change_password": u.MustChangePassword,
		"avatar_media_id":      nullableUUID(u.AvatarMediaID),
	}
}

func numericToFloat(v pgtype.Numeric) float64 {
	if !v.Valid {
		return 0
	}
	f, err := v.Float64Value()
	if err != nil {
		return 0
	}
	if !f.Valid {
		return 0
	}
	return f.Float64
}

func toInt64(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int32:
		return int64(x)
	case int:
		return int64(x)
	default:
		return 0
	}
}

func isHTTPS(s string) bool {
	return len(s) > 8 && s[:8] == "https://"
}

func floatToNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(floatToString(f))
	return n
}

func floatToString(f float64) string {
	neg := f < 0
	if neg {
		f = -f
	}
	i := int(f)
	frac := int((f - float64(i)) * 100)
	s := itoa(i) + "." + twoDigits(frac)
	if neg {
		s = "-" + s
	}
	return s
}

func twoDigits(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
