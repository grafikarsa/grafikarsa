package http

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgtype"
	"grafikarsa/api/internal/auth"
	"grafikarsa/api/internal/repo"
)

type ctxKey string

const (
	ctxUserID   ctxKey = "user_id"
	ctxRole     ctxKey = "role"
	ctxJTI      ctxKey = "jti"
	ctxPerms    ctxKey = "perms"
	ctxScope    ctxKey = "scope"
	ctxCSRF     ctxKey = "csrf_ok"
	ctxDeviceID ctxKey = "device_id"
)

func UserID(r *http.Request) string {
	v, _ := r.Context().Value(ctxUserID).(string)
	return v
}

func UserRole(r *http.Request) string {
	v, _ := r.Context().Value(ctxRole).(string)
	return v
}

func UserPerms(r *http.Request) map[string]bool {
	v, _ := r.Context().Value(ctxPerms).(map[string]bool)
	if v == nil {
		return map[string]bool{}
	}
	return v
}

func UserScope(r *http.Request) string {
	v, _ := r.Context().Value(ctxScope).(string)
	return v
}

// toJSONBytes marshal nilai ke JSON untuk disimpan di kolom JSONB.
//
// Error marshal sengaja tidak dipropagasi: JSONB tidak boleh kosong diam-diam
// tanpa jejak, jadi kalau gagal disimpan objek kosong yang valid ({} / [])
// supaya baris tetap terbaca dan bisa dibenerin. Caller yang butuh tahu
// status marshal harus memanggil json.Marshal langsung.
func toJSONBytes(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil || len(b) == 0 {
		return []byte("{}")
	}
	return b
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func ok(data any, r *http.Request) map[string]any {
	return map[string]any{
		"success":    true,
		"data":       data,
		"request_id": middleware.GetReqID(r.Context()),
	}
}

func okMeta(data, meta any, r *http.Request) map[string]any {
	return map[string]any{
		"success":    true,
		"data":       data,
		"meta":       meta,
		"request_id": middleware.GetReqID(r.Context()),
	}
}

func fail(code, message string, r *http.Request) map[string]any {
	return map[string]any{
		"success": false,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
		"request_id": middleware.GetReqID(r.Context()),
	}
}

func failDetails(code, message string, details []map[string]string, r *http.Request) map[string]any {
	return map[string]any{
		"success": false,
		"error": map[string]any{
			"code":    code,
			"message": message,
			"details": details,
		},
		"request_id": middleware.GetReqID(r.Context()),
	}
}

func setCookie(w http.ResponseWriter, name, value string, maxAge int, httpOnly bool, secure bool) {
	parts := []string{name + "=" + value, "Path=/", fmt.Sprintf("Max-Age=%d", maxAge), "SameSite=Lax"}
	if httpOnly {
		parts = append(parts, "HttpOnly")
	}
	if secure {
		parts = append(parts, "Secure")
	}
	w.Header().Add("Set-Cookie", strings.Join(parts, "; "))
}

func clearCookie(w http.ResponseWriter, name string, secure bool) {
	setCookie(w, name, "", -1, true, secure)
}

// CSRF token = HMAC(secret, hari + user_id). Terikat ke user (bukan global),
// tetap berotasi harian; terima H-1 agar sesi lintas tengah malam tetap jalan.
func newCSRFToken(secret, userID string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(time.Now().UTC().Format("2006-01-02") + ":" + userID))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func verifyCSRFToken(secret, userID, token string) bool {
	now := time.Now().UTC()
	for _, day := range []time.Time{now, now.Add(-24 * time.Hour)} {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(day.Format("2006-01-02") + ":" + userID))
		if hmac.Equal([]byte(token), []byte(base64.RawURLEncoding.EncodeToString(mac.Sum(nil)))) {
			return true
		}
	}
	return false
}

func toPgText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

func parseAccess(r *http.Request, secrets []string) *auth.AccessClaims {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return nil
	}
	claims, err := auth.ParseAccessToken(strings.TrimPrefix(h, "Bearer "), secrets)
	if err != nil {
		return nil
	}
	return claims
}

// clientIP: percaya HANYA x-client-ip yang di-set BFF (Caddy -> web -> BFF
// sudah memilih IP klien yang benar). X-Forwarded-For mentah dari klien tidak
// dipercaya supaya rate limit dan audit tidak bisa dipalsukan.
func clientIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("X-Client-IP")); ip != "" {
		return ip
	}
	// Fallback tanpa BFF: ambil entri TERAKHIR XFF (proxy menambahkan IP klien
	// di akhir), bukan yang pertama yang bisa dikirim klien.
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		return strings.TrimSpace(parts[len(parts)-1])
	}
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i > 0 {
		return host[:i]
	}
	return host
}

func audit(ctx context.Context, q *repo.Queries, userID, event, ip, ua string) {
	var uid pgtype.UUID
	if userID != "" {
		_ = uid.Scan(userID)
	}
	_ = q.InsertAuthAudit(ctx, repo.InsertAuthAuditParams{
		UserID:    uid,
		Event:     event,
		Ip:        toPgInet(ip),
		UserAgent: toPgText(ua),
	})
}

func toPgInet(s string) *netip.Addr { return toPgIP(s) }
