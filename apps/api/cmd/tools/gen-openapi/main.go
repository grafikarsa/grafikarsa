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

func writeResponses(b *strings.Builder, r route) {
	b.WriteString("      responses:\n")
	b.WriteString("        \"200\":\n")
	b.WriteString("          description: Berhasil\n")
	b.WriteString("          content:\n")
	b.WriteString("            application/json:\n")
	b.WriteString("              schema:\n")
	b.WriteString("                $ref: \"#/components/schemas/SuccessEnvelope\"\n")
	if r.method != "GET" {
		b.WriteString("        \"422\":\n          $ref: \"#/components/responses/ValidationError\"\n")
	}
	if r.csrf {
		b.WriteString("        \"403\":\n          $ref: \"#/components/responses/Forbidden\"\n")
	}
	if r.auth == "bearer" {
		b.WriteString("        \"401\":\n          $ref: \"#/components/responses/Unauthorized\"\n")
	}
	b.WriteString("        \"429\":\n          $ref: \"#/components/responses/TooManyRequests\"\n")
	b.WriteString("        \"500\":\n          $ref: \"#/components/responses/InternalError\"\n")
}
