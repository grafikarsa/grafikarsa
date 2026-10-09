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

func publicUser(u repo.GetUserByUsernameRow) map[string]any {
	return map[string]any{
		"id": u.Username, "username": u.Username,
		"name": u.Name, "email": u.Email,
		"role_primary": u.RolePrimary, "status": u.Status,
	}
}
