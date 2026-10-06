package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ramon/trackline/internal/config"
)

func TestEmbeddingAndCrossSiteCookie(t *testing.T) {
	app := &App{Config: config.Config{SecureCookies: true}}
	recorder := httptest.NewRecorder()
	app.headers(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		app.cookie(w, "session", "token", 60)
	})).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if got := recorder.Header().Get("Content-Security-Policy"); !strings.Contains(got, "frame-ancestors https://hub.cds.com.br") {
		t.Fatalf("HubCDS missing from frame-ancestors: %q", got)
	}
	if got := recorder.Header().Get("Set-Cookie"); !strings.Contains(got, "SameSite=None") || !strings.Contains(got, "Secure") {
		t.Fatalf("cross-site cookie is not secure: %q", got)
	}
}
