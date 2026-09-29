package updatecheck

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestNewer(t *testing.T) {
	for _, test := range []struct {
		latest, current string
		want            bool
	}{{"0.1.1", "0.1.0", true}, {"1.0.0", "0.9.9", true}, {"0.1.0", "0.1.0", false}, {"0.1.0", "0.2.0", false}} {
		got, err := newer(test.latest, test.current)
		if err != nil {
			t.Fatal(err)
		}
		if got != test.want {
			t.Fatalf("newer(%q, %q) = %v, want %v", test.latest, test.current, got, test.want)
		}
	}
}

func TestCheckFindsRelease(t *testing.T) {
	checker := New("https://api.github.test/latest", "0.1.0")
	checker.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("User-Agent") != "1984-update-checker" {
			t.Error("missing user agent")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"tag_name":"v0.2.0","html_url":"https://github.example/releases/v0.2.0"}`)), Header: make(http.Header)}, nil
	})}
	if err := checker.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	update := checker.Snapshot()
	if !update.Available || update.Latest != "0.2.0" {
		t.Fatalf("unexpected update: %#v", update)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}
