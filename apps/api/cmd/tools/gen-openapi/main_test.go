package main

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"go/ast"
)

func loadRoutes(t *testing.T) []route {
	t.Helper()
	fset := token.NewFileSet()
	path := routesPath()
	file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	w := &walker{}
	ast.Walk(w, file)
	if len(w.routes) == 0 {
		t.Fatal("tidak ada route terbaca")
	}
	return dedupeRoutes(w.routes)
}

// Kontrak tidak boleh bocor: setiap route harus punya klasifikasi security.
func TestSemuaRoutePunyaSecurity(t *testing.T) {
	for _, r := range loadRoutes(t) {
		switch r.auth {
		case "public", "optional", "bearer":
		default:
			t.Errorf("%s %s: security tidak terklasifikasi (%q)", r.method, r.path, r.auth)
		}
	}
}

// Semua mutasi wajib CSRF, kecuali tiga endpoint yang memang tidak punya
// sesi/cookie untuk diverifikasi (login, refresh, feedback guest).
func TestMutasiWajibCSRF(t *testing.T) {
	// Pengecualian yang disengaja:
	// - login/refresh: belum ada sesi untuk diverifikasi (login pakai Turnstile
	//   saat brute-force, refresh memakai cookie + SameSite).
	// - feedback: guest tanpa cookie; dilindungi Turnstile + rate limit.
	// - search POST: endpoint BACA (tidak mengubah state) yang memakai POST
	//   agar query panjang/UTF-8 aman; CSRF tidak relevan untuk data publik.
	allowed := map[string]bool{
		"/api/v1/auth/login":   true,
		"/api/v1/auth/refresh": true,
		"/api/v1/feedback":     true,
		"/api/v1/search":       true,
	}
	for _, r := range loadRoutes(t) {
		if r.method == "GET" || r.csrf || allowed[r.path] {
			continue
		}
		t.Errorf("%s %s: mutasi tanpa CSRF dan bukan pengecualian", r.method, r.path)
	}
}

// Endpoint admin harus punya permission eksplisit (tidak boleh hanya "login").
func TestAdminPunyaPermission(t *testing.T) {
	for _, r := range loadRoutes(t) {
		if !strings.Contains(r.path, "/admin/") {
			continue
		}
		if len(r.perms) == 0 {
			t.Errorf("%s %s: endpoint admin tanpa x-permissions", r.method, r.path)
		}
	}
}

// Path harus konsisten berawalan /api dan tidak ada duplikat method+path.
func TestPathKonsisten(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range loadRoutes(t) {
		if !strings.HasPrefix(r.path, "/api/") {
			t.Errorf("path tidak berawalan /api: %s", r.path)
		}
		if strings.Contains(r.path, "//") {
			t.Errorf("path mengandung //: %s", r.path)
		}
		key := r.method + " " + r.path
		if seen[key] {
			t.Errorf("duplikat route: %s", key)
		}
		seen[key] = true
	}
}

// Generator harus deterministik dan menghasilkan YAML yang memuat komponen
// wajib (envelope + security scheme) serta jumlah path yang sama dengan route.
func TestRenderDeterministik(t *testing.T) {
	routes := loadRoutes(t)
	a := render(routes)
	b := render(routes)
	if a != b {
		t.Fatal("output generator tidak deterministik")
	}
	for _, want := range []string{"openapi: 3.1.0", "bearerAuth:", "SuccessEnvelope:", "ErrorEnvelope:", "paths:"} {
		if !strings.Contains(a, want) {
			t.Errorf("komponen wajib hilang dari kontrak: %q", want)
		}
	}
	for _, r := range routes {
		if !strings.Contains(a, r.path) {
			t.Errorf("path %s hilang dari kontrak", r.path)
		}
	}
}
