// Command gen-openapi menghasilkan packages/contracts/openapi.yaml dari
// sumber kebenaran kode (routes.go) memakai parser AST sehingga nesting
// r.Route / r.Group / r.With dihitung presisi — kontrak tidak bisa "stale"
// karena CI meregenerasi file ini dan membandingkan hasilnya.
//
// Jalankan: make openapi
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

const routesRel = "internal/http/routes.go"

// routesPath: resolve dari lokasi file sumber, sehingga generator tetap benar
// baik dijalankan dari root module (`go run ./cmd/tools/gen-openapi`) maupun
// dari `go test` di dalam direktori paketnya.
func routesPath() string {
	if _, err := os.Stat(routesRel); err == nil {
		return routesRel
	}
	if _, file, _, ok := runtime.Caller(0); ok {
		dir := filepath.Dir(file)
		for i := 0; i < 5; i++ {
			cand := filepath.Join(dir, routesRel)
			if _, err := os.Stat(cand); err == nil {
				return cand
			}
			dir = filepath.Dir(dir)
		}
	}
	return routesRel
}

type route struct {
	method  string
	path    string
	handler string
	auth    string // public | optional | bearer
	perms   []string
	csrf    bool
}

type frame struct {
	prefix string
	auth   string
	perms  []string
}

var httpMethods = map[string]string{
	"Get": "GET", "Post": "POST", "Put": "PUT", "Patch": "PATCH", "Delete": "DELETE",
}

type walker struct {
	stack  []frame
	routes []route
}

func main() {
	path := routesPath()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		fail("parse %s: %v", path, err)
	}
	w := &walker{}
	ast.Walk(w, file)
	if len(w.routes) == 0 {
		fail("tidak ada route terbaca dari %s", path)
	}
	os.Stdout.WriteString(render(dedupeRoutes(w.routes)))
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func dedupeRoutes(in []route) []route {
	seen := map[string]bool{}
	var out []route
	for _, r := range in {
		key := r.method + " " + r.path
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, r)
	}
	return out
}

// Visit: r.Route(...) dan r.Group(...) menurunkan context (prefix + middleware)
// hanya untuk subtree-nya, lalu context di-pop saat subtree selesai.
func (w *walker) Visit(n ast.Node) ast.Visitor {
	switch x := n.(type) {
	case *ast.FuncDecl:
		if x.Name != nil && x.Name.Name == "routes" {
			return w
		}
		return nil
	case *ast.CallExpr:
		if prefix, ok := routePrefix(x); ok {
			w.walkGroup(x, frame{prefix: prefix})
			return nil
		}
		if isGroupCall(x) {
			w.walkGroup(x, frame{})
			return nil
		}
		if mw := middlewareUse(x); mw != "" {
			if len(w.stack) > 0 {
				top := &w.stack[len(w.stack)-1]
				switch mw {
				case "authRequired":
					top.auth = "bearer"
				case "authOptional":
					if top.auth != "bearer" {
						top.auth = "optional"
					}
				default:
					top.perms = append(top.perms, strings.Split(mw, ",")...)
				}
			}
			return w
		}
		if r, ok := w.httpRoute(x); ok {
			w.routes = append(w.routes, r)
		}
	}
	return w
}

// walkGroup: telusuri subtree r.Route/r.Group secara manual supaya context
// (prefix + middleware) hanya aktif selama subtree tersebut.
func (w *walker) walkGroup(call *ast.CallExpr, f frame) {
	w.stack = append(w.stack, f)
	for _, a := range call.Args {
		ast.Walk(w, a)
	}
	w.stack = w.stack[:len(w.stack)-1]
}

func routePrefix(call *ast.CallExpr) (string, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Route" || len(call.Args) == 0 {
		return "", false
	}
	return stringLit(call.Args[0])
}

func isGroupCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "Group"
}

// middlewareUse mengenali dua bentuk argumen r.Use:
//
//	r.Use(a.authRequired)                -> method value (SelectorExpr)
//	r.Use(a.requirePerm("users.view"))   -> call (CallExpr)
func middlewareUse(call *ast.CallExpr) string {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Use" || len(call.Args) == 0 {
		return ""
	}
	switch arg := call.Args[0].(type) {
	case *ast.SelectorExpr:
		switch arg.Sel.Name {
		case "authRequired":
			return "authRequired"
		case "authOptional":
			return "authOptional"
		}
	case *ast.CallExpr:
		is, ok := arg.Fun.(*ast.SelectorExpr)
		if !ok {
			return ""
		}
		switch is.Sel.Name {
		case "requirePerm", "withPerm":
			if len(arg.Args) == 1 {
				if v, ok := stringLit(arg.Args[0]); ok {
					return v
				}
			}
		case "requireAnyPerm":
			var perms []string
			for _, a := range arg.Args {
				if v, ok := stringLit(a); ok {
					perms = append(perms, v)
				}
			}
			return strings.Join(perms, ",")
		}
	}
	return ""
}

func stringLit(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return s, true
}

func (w *walker) httpRoute(call *ast.CallExpr) (route, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return route{}, false
	}
	method, isHTTP := httpMethods[sel.Sel.Name]
	if !isHTTP || len(call.Args) == 0 {
		return route{}, false
	}
	p, ok := stringLit(call.Args[0])
	if !ok {
		return route{}, false
	}

	full := "/api"
	for _, f := range w.stack {
		full += f.prefix
	}
	full += p
	for strings.Contains(full, "//") {
		full = strings.ReplaceAll(full, "//", "/")
	}
	if len(full) > 1 && strings.HasSuffix(full, "/") {
		full = strings.TrimSuffix(full, "/")
	}

	// Endpoint probe di root API (/health, /ready, /metrics) BUKAN
	// bagian kontrak publik: BFF tidak menyajikannya, dan probe container
	// memanggilnya langsung ke API. Mendokumentasikan mereka sebagai
	// "/api/health" berarti promising path yang tidak ada.
	if skipPublicPath(full) {
		return route{}, false
	}

	r := route{method: method, path: full}
	for _, f := range w.stack {
		switch f.auth {
		case "bearer":
			r.auth = "bearer"
		case "optional":
			if r.auth == "" {
				r.auth = "optional"
			}
		}
		r.perms = append(r.perms, f.perms...)
	}

	ast.Inspect(call, func(n ast.Node) bool {
		inner, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		is, ok := inner.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		switch is.Sel.Name {
		case "authRequired":
			r.auth = "bearer"
		case "authOptional":
			if r.auth == "" {
				r.auth = "optional"
			}
		case "csrf":
			r.csrf = true
		case "requirePerm", "withPerm":
			if len(inner.Args) == 1 {
				if s, ok := stringLit(inner.Args[0]); ok {
					r.perms = append(r.perms, s)
				}
			}
		case "requireAnyPerm":
			for _, a := range inner.Args {
				if s, ok := stringLit(a); ok {
					r.perms = append(r.perms, s)
				}
			}
		default:
			if r.handler == "" && strings.HasPrefix(is.Sel.Name, "handle") {
				r.handler = is.Sel.Name
			}
		}
		return true
	})
	if r.handler == "" {
		r.handler = "inline"
	}
	if r.auth == "" {
		r.auth = "public"
	}
	r.perms = dedupe(r.perms)
	return r, true
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] && s != "" {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

func tagOf(path string) string {
	trimmed := path
	for _, prefix := range []string{"/api/v1/", "/api/"} {
		if strings.HasPrefix(path, prefix) {
			trimmed = strings.TrimPrefix(path, prefix)
			break
		}
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) == 0 || parts[0] == "" {
		return "system"
	}
	switch parts[0] {
	case "auth":
		return "auth"
	case "me":
		return "profile"
	case "admin":
		if len(parts) > 1 {
			return "admin-" + parts[1]
		}
		return "admin"
	case "health", "ready", "metrics":
		return "system"
	default:
		return parts[0]
	}
}

func opID(r route) string {
	clean := strings.NewReplacer("/", "_", "{", "", "}", "", "-", "_", ":", "_").Replace(strings.TrimPrefix(r.path, "/api/v1/"))
	clean = strings.Trim(strings.TrimPrefix(clean, "api_"), "_")
	return strings.ToLower(r.method) + "_" + clean
}

func summary(r route) string {
	scope := "publik"
	switch r.auth {
	case "bearer":
		scope = "butuh login"
	case "optional":
		scope = "opsional login"
	}
	if len(r.perms) > 0 {
		scope += ", permission: " + strings.Join(r.perms, "|")
	}
	if r.csrf {
		scope += ", wajib CSRF"
	}
	return fmt.Sprintf("%s (%s)", r.handler, scope)
}

func render(routes []route) string {
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].path == routes[j].path {
			return routes[i].method < routes[j].method
		}
		return routes[i].path < routes[j].path
	})

	var b strings.Builder
	b.WriteString("# Kode ini di-generate dari apps/api/internal/http/routes.go.\n")
	b.WriteString("# JANGAN edit manual — jalankan `make openapi`.\n")
	b.WriteString("# CI memverifikasi tidak ada drift (.github/workflows/ci.yml).\n")
	b.WriteString("openapi: 3.1.0\n")
	b.WriteString("info:\n")
	b.WriteString("  title: Grafikarsa API\n")
	b.WriteString("  version: 2.0.0\n")
	b.WriteString("  description: |\n")
	b.WriteString("    API katalog portfolio + jaringan sosial SMKN 4 Malang.\n")
	b.WriteString("    Browser memanggil lewat BFF Next.js (/api/*); backend tidak diekspos publik.\n")
	b.WriteString("    Autentikasi: Bearer access token (15 menit, disimpan di memori FE) +\n")
	b.WriteString("    cookie refresh __Host-rt HttpOnly. Mutasi wajib memakai X-CSRF-Token\n")
	b.WriteString("    yang cocok dengan cookie __Host-csrf (double-submit, terikat ke user).\n")
	b.WriteString("    Permission ditulis sebagai x-permissions; admin melewati daftar itu,\n")
	b.WriteString("    admin_jurusan dibatasi row-level scope (claim scope_major_id).\n")
	b.WriteString("servers:\n")
	b.WriteString("  - url: /\n")
	b.WriteString("    description: Lewat BFF (produksi)\n")
	b.WriteString("  - url: http://localhost:8091\n")
	b.WriteString("    description: API langsung (hanya dev/jaringan internal)\n")

	tags := map[string]bool{}
	for _, r := range routes {
		tags[tagOf(r.path)] = true
	}
	var tagList []string
	for t := range tags {
		tagList = append(tagList, t)
	}
	sort.Strings(tagList)
	b.WriteString("tags:\n")
	for _, t := range tagList {
		fmt.Fprintf(&b, "  - name: %s\n", t)
	}

	b.WriteString("paths:\n")
	byPath := map[string][]route{}
	var order []string
	for _, r := range routes {
		if _, ok := byPath[r.path]; !ok {
			order = append(order, r.path)
		}
		byPath[r.path] = append(byPath[r.path], r)
	}
	sort.Strings(order)
	for _, p := range order {
		fmt.Fprintf(&b, "  %q:\n", p)
		for _, r := range byPath[p] {
			fmt.Fprintf(&b, "    %s:\n", strings.ToLower(r.method))
			fmt.Fprintf(&b, "      tags: [%s]\n", tagOf(r.path))
			fmt.Fprintf(&b, "      operationId: %s\n", opID(r))
			fmt.Fprintf(&b, "      summary: %q\n", summary(r))
			fmt.Fprintf(&b, "      x-handler: %s\n", r.handler)
			switch r.auth {
			case "bearer":
				b.WriteString("      security:\n        - bearerAuth: []\n")
			case "optional":
				b.WriteString("      security:\n        - bearerAuth: []\n        - {}\n")
			default:
				b.WriteString("      security: []\n")
			}
			if len(r.perms) > 0 {
				fmt.Fprintf(&b, "      x-permissions: [%s]\n", strings.Join(r.perms, ", "))
			}
			if r.csrf {
				b.WriteString("      x-csrf: required\n")
			}
			writePathParams(&b, r.path)
			writeRequiredQueryParams(&b, r.path)
			writeRequestBody(&b, r.method, r.path)
			writeResponses(&b, r)
		}
	}
	b.WriteString(componentsYAML)
	out := b.String()
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out
}

// writeResponses menulis blok responses untuk satu operasi.
//
// Status code yang benar-benar dikembalikan handler tapi sebelumnya TIDAK
// terdokumentasikan: 400 (body/parameter rusak), 404 (sumber daya tidak ada),
// 409 (duplikat), 201 (dibuat), dan 403 (di luar scope / permission
// ditolak — sebelumnya hanya ditulis untuk route yang butuh CSRF).
//
// Driver: schemathesis di CI menandai "undocumented status code" kalau status
// yang benar-benar dikembalikan handler tidak ada di spec.
func writeResponses(b *strings.Builder, r route) {
	b.WriteString("      responses:\n")
	if r.method == "POST" {
		b.WriteString("        \"201\":\n          description: Berhasil dibuat\n")
		b.WriteString("          content:\n            application/json:\n              schema:\n                $ref: \"#/components/schemas/SuccessEnvelope\"\n")
	}
	b.WriteString("        \"200\":\n")
	b.WriteString("          description: Berhasil\n")
	b.WriteString("          content:\n            application/json:\n              schema:\n                $ref: \"#/components/schemas/SuccessEnvelope\"\n")
	b.WriteString("        \"400\":\n          $ref: \"#/components/responses/BadRequest\"\n")
	// Redirect 301: hanya untuk route "/storage/*" yang meneruskan URL lama
	// v1 ke media baru (PRD §16.1, berlaku 6 bulan pasca go-live).
	if strings.Contains(r.path, "/storage/") {
		b.WriteString("        \"301\":\n          description: Permanen dialihkan ke media baru\n")
		b.WriteString("          headers:\n            Location: { type: string }\n")
		// 410: URL lama terdaftar tapi file-nya sudah tidak tersedia
		// (setelah masa redirect 6 bulan berakhir).
		b.WriteString("        \"410\":\n          $ref: \"#/components/responses/Gone\"\n")
	}
	if canReturn404(r.path) {
		b.WriteString("        \"404\":\n          $ref: \"#/components/responses/NotFound\"\n")
	}
	if r.method != "GET" {
		b.WriteString("        \"422\":\n          $ref: \"#/components/responses/ValidationError\"\n")
	}
	if r.csrf || len(r.perms) > 0 {
		b.WriteString("        \"403\":\n          $ref: \"#/components/responses/Forbidden\"\n")
	}
	if r.auth == "bearer" {
		b.WriteString("        \"401\":\n          $ref: \"#/components/responses/Unauthorized\"\n")
	}
	b.WriteString("        \"409\":\n          $ref: \"#/components/responses/Conflict\"\n")
	// 405: router mengembalikan 405 kalau path-nya dikenali tapi method-nya
	// tidak (mis. GET ke endpoint yang hanya menerima POST). Tanpa ini,
	// contract test menandai "Wrong response to undeclared method" —
	// padahal respons 405 itu benar.
	b.WriteString("        \"405\":\n          $ref: \"#/components/responses/MethodNotAllowed\"\n")
	b.WriteString("        \"429\":\n          $ref: \"#/components/responses/TooManyRequests\"\n")
	b.WriteString("        \"500\":\n          $ref: \"#/components/responses/InternalError\"\n")
}

// canReturn404 menentukan apakah sebuah endpoint bisa membalas 404.
//
// Dua sumber 404:
//  1. Endpoint dengan path parameter — "/portfolios/{id}" bisa 404 kalau
//     id-nya tidak ada.
//  2. Endpoint "singleton" yang mewakili satu entitas dan belum tentu ada:
//     "/changelogs/latest" (404 kalau belum ada changelog), "/media/by-key/*"
//     (404 kalau key-nya tidak dikenal), "/storage/*" (redirect ke media lama).
//
// Selain itu, endpoint koleksi tidak pernah membalas 404 — daftarnya kosong
// tetap 200 dengan array kosong.
func canReturn404(path string) bool {
	if len(pathParams(path)) > 0 {
		return true
	}
	for _, frag := range singletonPaths {
		if strings.Contains(path, frag) {
			return true
		}
	}
	return false
}

// singletonPaths: endpoint yang membalas 404 kalau entitasnya belum ada.
var singletonPaths = []string{
	"/changelogs/latest",
	"/media/by-key/",
	"/storage/",
	// by-slug: 404 kalau kombinasi username+slug-nya tidak ada.
	"/portfolios/by-slug",
}

// skipPublicPath menentukan apakah sebuah path bukan bagian kontrak publik.
//
// Endpoint probe di root API (/health, /ready, /metrics) dibaca probe
// container langsung ke API, bukan lewat BFF — jadi "/api/health" adalah
// path yang tidak pernah ada. Mendokumentasikannya berarti promising
// sesuatu yang tidak bisa dipanggil.
func skipPublicPath(path string) bool {
	switch path {
	case "/api/health", "/api/ready", "/api/metrics":
		return true
	}
	return false
}

// requestBodies: deskripsi body request untuk endpoint yang punya body.
//
// Generator tidak memodelkan body secara umum, jadi endpoint yang butuh body
// didaftarkan di sini.
//
// Penting: PRD §5 menyebut "POST /search — query dikirim di body agar
// panjang/UTF-8 aman", dan endpoint itu memang menerima body JSON
// {query|q}. Tanpa deskripsi body di spec, klien hasil generate tidak bisa
// merepresentasikan request itu — dan contract test menganggap request yang
// tidak sesuai spec "ditolak".
//
// Kuncinya "METHOD path" — bukan path saja: /search punya GET (query di URL)
// dan POST (query di body). Kalau requestBody ditempel ke pathnya, GET juga
// diklaim punya body — dan schemathesis lalu mengirim body ke GET yang
// diaplikasinya mengabaikan, sehingga request itu dianggap melanggar skema
// padahal server tidak pernah memcion body itu.
var requestBodies = map[string]string{
	"POST /api/v1/search": `      requestBody:
        required: false
        content:
          application/json:
            schema:
              type: object
              properties:
                query: { type: string, minLength: 2 }
                q: { type: string, minLength: 2 }
          text/plain:
            schema: { type: string }
`,
}

// writeRequestBody menulis blok requestBody untuk operasi yang punya body.
func writeRequestBody(b *strings.Builder, method, path string) {
	body, ok := requestBodies[method+" "+path]
	if !ok {
		return
	}
	b.WriteString(body)
}

// pathParams mengembalikan daftar nama parameter path pada sebuah path,
// mis. "/api/v1/portfolios/id/{id}" -> ["id"].
//
// Path parameter WAJIB dideklarasikan di OpenAPI. Tanpa itu spec-nya tidak
// valid dan tooling (validator, client generator) menolaknya.
func pathParams(path string) []string {
	var out []string
	for _, seg := range strings.Split(path, "/") {
		if len(seg) > 2 && strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			out = append(out, seg[1:len(seg)-1])
		}
	}
	return out
}

// uuidPathParams: nama parameter yang isinya UUID.
var uuidPathParams = map[string]bool{
	"id": true, "blockId": true, "userId": true, "portfolio_id": true,
}

// queryParamsRequired: parameter query WAJIB per endpoint.
//
// Generator tidak memodelkan parameter query secara umum, jadi endpoint
// yang butuh query wajib didaftarkan di sini. Tanpa ini, spec-nya
// tidak memberi tahu klien parameter mana yang wajib diisi — dan contract
// test menandai "Schema-valid request rejected" (permintaan yang valid
// menurut spec ditolak server karena parameter wajibnya tidak ada).
var queryParamsRequired = map[string][]struct {
	Name      string
	Type      string
	MinLength int
	Pattern   string
}{
	//nolint:govet // peta dengan struct inline; nama field sudah jelas
	"/api/v1/portfolios/by-slug": {
		{Name: "username", Type: "string", MinLength: 1},
		{Name: "slug", Type: "string", MinLength: 1},
	},
	"/api/v1/search": {
		// Server menolak query < 2 karakter ATAU yang tidak mengandung
		// karakter alfanumerik (supaya "%%" tidak mengembalikan seluruh
		// katalog). Pola di spec harus mencerminkan aturan itu.
		{Name: "q", Type: "string", MinLength: 2, Pattern: ".*[a-zA-Z0-9].*"},
	},
}

// writeRequiredQueryParams menulis blok parameter query wajib.
func writeRequiredQueryParams(b *strings.Builder, path string) {
	params, ok := queryParamsRequired[path]
	if !ok {
		return
	}
	b.WriteString("      parameters:\n")
	for _, p := range params {
		fmt.Fprintf(b, "        - name: %s\n          in: query\n          required: true\n", p.Name)
		// minLength wajib: tanpa itu skema membolehkan nilai yang server
		// tolak (string kosong, query terlalu pendek), jadi kontrak
		// bertentangan dengan perilaku server.
		min := p.MinLength
		if min <= 0 {
			min = 1
		}
		if p.Pattern != "" {
			fmt.Fprintf(b, "          schema: { type: %s, minLength: %d, pattern: %q }\n", p.Type, min, p.Pattern)
		} else {
			fmt.Fprintf(b, "          schema: { type: %s, minLength: %d }\n", p.Type, min)
		}
	}
}

// pathParamSchemas: skema parameter path per nama.
//
// Penting: parameter path harus punya POLA, bukan `type: string` bebas.
// Kalau skema membolehkan string apa pun, schemathesis menganggap request
// dengan username gibberish itu "valid menurut spec" — padahal server
// menolak dengan 400. Kontraknya jadi bertentangan dengan dirinya sendiri.
//
// Parameter UUID tetap `format: uuid`; sisanya punya pola yang sama dengan
// validasi di server.
var pathParamSchemas = map[string]string{
	"username": `{ type: string, pattern: "^[a-z0-9._-]{3,30}$" }`,
	"slug":     `{ type: string, pattern: "^[a-z0-9._-]{3,80}$" }`,
	"version":  `{ type: string, pattern: "^[0-9]+\\.[0-9]+$" }`,
	"key":      `{ type: string, minLength: 1 }`,
}

// writePathParams menulis blok parameters untuk path yang punya {param}.
func writePathParams(b *strings.Builder, path string) {
	params := pathParams(path)
	if len(params) == 0 {
		return
	}
	b.WriteString("      parameters:\n")
	for _, p := range params {
		fmt.Fprintf(b, "        - name: %s\n          in: path\n          required: true\n", p)
		switch {
		case uuidPathParams[p]:
			b.WriteString("          schema: { type: string, format: uuid }\n")
		case pathParamSchemas[p] != "":
			fmt.Fprintf(b, "          schema: %s\n", pathParamSchemas[p])
		default:
			b.WriteString("          schema: { type: string }\n")
		}
	}
}
