package authctx

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
)

func TestPrincipalRoundTrip(t *testing.T) {
	want := &auth.Principal{UserID: uuid.New(), DisplayName: "Siswa Uji"}

	if got := Principal(WithPrincipal(context.Background(), want)); got != want {
		t.Fatalf("Principal() = %v, want the stored principal", got)
	}
}

func TestPrincipalAnonymousIsNil(t *testing.T) {
	if got := Principal(context.Background()); got != nil {
		t.Fatalf("Principal() = %v, want nil", got)
	}
}

func TestMetaRoundTrip(t *testing.T) {
	want := Meta{IP: "127.0.0.1", UserAgent: "uji/1.0"}

	if got := MetaFrom(WithMeta(context.Background(), want)); got != want {
		t.Fatalf("MetaFrom() = %+v, want %+v", got, want)
	}
}

func TestMetaFromEmptyContext(t *testing.T) {
	if got := MetaFrom(context.Background()); got != (Meta{}) {
		t.Fatalf("MetaFrom() = %+v, want zero value", got)
	}
}

func TestJarCookiesAreACopy(t *testing.T) {
	jar := &Jar{}
	jar.Add(&http.Cookie{Name: "a", Value: "1"})
	jar.Add(&http.Cookie{Name: "b", Value: "2"})

	cookies := jar.Cookies()
	if len(cookies) != 2 {
		t.Fatalf("Cookies() length = %d, want 2", len(cookies))
	}

	cookies[0] = nil
	if got := jar.Cookies()[0]; got == nil || got.Name != "a" {
		t.Fatalf("Cookies() exposes the internal slice: %v", got)
	}
}

func TestJarIgnoresNilCookie(t *testing.T) {
	jar := &Jar{}
	jar.Add(nil)

	if got := len(jar.Cookies()); got != 0 {
		t.Fatalf("Cookies() length = %d, want 0", got)
	}
}

func TestSetCookieWithoutJarIsNoOp(t *testing.T) {
	SetCookie(context.Background(), &http.Cookie{Name: "a", Value: "1"})
}

func TestSetCookieQueuesInJar(t *testing.T) {
	jar := &Jar{}
	ctx := WithJar(context.Background(), jar)

	SetCookie(ctx, &http.Cookie{Name: "a", Value: "1"})

	if got := jar.Cookies(); len(got) != 1 || got[0].Name != "a" {
		t.Fatalf("Cookies() = %v, want one queued cookie", got)
	}
}
