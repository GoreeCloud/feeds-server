package retrieval

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type staticResolver map[string][]netip.Addr

func (r staticResolver) LookupNetIP(_ context.Context, _ string, host string) ([]netip.Addr, error) {
	values, ok := r[host]
	if !ok {
		return nil, errors.New("host not found")
	}
	return append([]netip.Addr(nil), values...), nil
}

func loopbackPrefix() netip.Prefix {
	return netip.MustParsePrefix("127.0.0.0/8")
}

func TestDefaultPolicyRejectsLoopback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("request reached blocked loopback server")
	}))
	defer server.Close()

	fetcher, err := New(Config{MaxAttempts: 1})
	if err != nil {
		t.Fatal(err)
	}
	_, err = fetcher.Fetch(context.Background(), server.URL, Conditions{})
	if !errors.Is(err, ErrBlockedDestination) {
		t.Fatalf("Fetch() error = %v, want blocked destination", err)
	}
}

func TestFetchConditionalMetadataAndHeaders(t *testing.T) {
	var observed bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observed = true
		if got := r.Header.Get("User-Agent"); got != defaultUserAgent {
			t.Fatalf("User-Agent = %q", got)
		}
		if got := r.Header.Get("If-None-Match"); got != "\"old\"" {
			t.Fatalf("If-None-Match = %q", got)
		}
		if got := r.Header.Get("If-Modified-Since"); got != "Mon, 01 Jan 2024 00:00:00 GMT" {
			t.Fatalf("If-Modified-Since = %q", got)
		}
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Header().Set("ETag", "\"new\"")
		w.Header().Set("Last-Modified", "Tue, 02 Jan 2024 00:00:00 GMT")
		_, _ = w.Write([]byte("<rss/>"))
	}))
	defer server.Close()

	fetcher, err := New(Config{AllowCIDRs: []netip.Prefix{loopbackPrefix()}, MaxAttempts: 1})
	if err != nil {
		t.Fatal(err)
	}
	result, err := fetcher.Fetch(context.Background(), server.URL, Conditions{
		ETag:         "\"old\"",
		LastModified: "Mon, 01 Jan 2024 00:00:00 GMT",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !observed {
		t.Fatal("server did not receive request")
	}
	if string(result.Body) != "<rss/>" || result.ETag != "\"new\"" ||
		result.LastModified != "Tue, 02 Jan 2024 00:00:00 GMT" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestNotModified(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") != "\"same\"" {
			t.Fatal("conditional ETag missing")
		}
		w.Header().Set("ETag", "\"same\"")
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	fetcher, err := New(Config{AllowCIDRs: []netip.Prefix{loopbackPrefix()}, MaxAttempts: 1})
	if err != nil {
		t.Fatal(err)
	}
	result, err := fetcher.Fetch(context.Background(), server.URL, Conditions{ETag: "\"same\""})
	if err != nil {
		t.Fatal(err)
	}
	if !result.NotModified || result.StatusCode != http.StatusNotModified || len(result.Body) != 0 {
		t.Fatalf("unexpected 304 result: %+v", result)
	}
}

func TestResponseBodyLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 64)))
	}))
	defer server.Close()

	fetcher, err := New(Config{
		AllowCIDRs:   []netip.Prefix{loopbackPrefix()},
		MaxBodyBytes: 16,
		MaxAttempts:  1,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = fetcher.Fetch(context.Background(), server.URL, Conditions{})
	if !errors.Is(err, ErrBodyTooLarge) {
		t.Fatalf("Fetch() error = %v, want body too large", err)
	}
}

func TestRetryableStatusUsesBoundedBackoff(t *testing.T) {
	var requests int32
	var sleeps int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, *http.Request) {
		count := atomic.AddInt32(&requests, 1)
		if count < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("<rss/>"))
	}))
	defer server.Close()

	fetcher, err := New(Config{
		AllowCIDRs:  []netip.Prefix{loopbackPrefix()},
		MaxAttempts: 3,
		BaseBackoff: time.Millisecond,
		MaxBackoff:  2 * time.Millisecond,
		Sleep: func(context.Context, time.Duration) error {
			atomic.AddInt32(&sleeps, 1)
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := fetcher.Fetch(context.Background(), server.URL, Conditions{})
	if err != nil {
		t.Fatal(err)
	}
	if result.StatusCode != http.StatusOK || requests != 3 || sleeps != 2 {
		t.Fatalf("status=%d requests=%d sleeps=%d", result.StatusCode, requests, sleeps)
	}
}

func TestRedirectDestinationIsRevalidated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, *http.Request) {
		http.Redirect(w, &http.Request{}, "http://blocked.test/latest", http.StatusFound)
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	resolver := staticResolver{
		"allowed.test": {netip.MustParseAddr("127.0.0.1")},
		"blocked.test": {netip.MustParseAddr("169.254.169.254")},
	}
	fetcher, err := New(Config{
		AllowCIDRs:  []netip.Prefix{loopbackPrefix()},
		Resolver:    resolver,
		MaxAttempts: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = fetcher.Fetch(context.Background(), "http://allowed.test:"+serverURL.Port(), Conditions{})
	if !errors.Is(err, ErrBlockedDestination) {
		t.Fatalf("Fetch() error = %v, want blocked redirect destination", err)
	}
}

func TestHTTPSDowngradeRedirectRejected(t *testing.T) {
	fetcher, err := New(Config{AllowCIDRs: []netip.Prefix{loopbackPrefix()}})
	if err != nil {
		t.Fatal(err)
	}
	previous := httptest.NewRequest(http.MethodGet, "https://127.0.0.1/original", nil)
	next := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/redirected", nil)
	err = fetcher.checkRedirect(next, []*http.Request{previous})
	if !errors.Is(err, ErrDowngradeRedirect) {
		t.Fatalf("checkRedirect() error = %v, want downgrade rejection", err)
	}
}

func TestEmbeddedCredentialsRejected(t *testing.T) {
	fetcher, err := New(Config{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = fetcher.Fetch(context.Background(), "https://user:secret@example.com/feed.xml", Conditions{})
	if !errors.Is(err, ErrInvalidURL) {
		t.Fatalf("Fetch() error = %v, want invalid URL", err)
	}
}

func TestMixedDNSAnswersFailClosed(t *testing.T) {
	fetcher, err := New(Config{Resolver: staticResolver{
		"mixed.test": {
			netip.MustParseAddr("8.8.8.8"),
			netip.MustParseAddr("127.0.0.1"),
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	target, err := url.Parse("https://mixed.test/feed")
	if err != nil {
		t.Fatal(err)
	}
	if err := fetcher.validateTarget(context.Background(), target); !errors.Is(err, ErrBlockedDestination) {
		t.Fatalf("validateTarget() error = %v, want blocked destination", err)
	}
}

func TestRemoteAddressRevalidated(t *testing.T) {
	fetcher, err := New(Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := fetcher.validateRemoteAddr(&net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 80}); !errors.Is(err, ErrBlockedDestination) {
		t.Fatalf("validateRemoteAddr() error = %v, want blocked destination", err)
	}
}

func TestFinalNonSuccessReturnsStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	fetcher, err := New(Config{AllowCIDRs: []netip.Prefix{loopbackPrefix()}, MaxAttempts: 1})
	if err != nil {
		t.Fatal(err)
	}
	result, err := fetcher.Fetch(context.Background(), server.URL, Conditions{})
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusNotFound {
		t.Fatalf("Fetch() error = %v, want HTTP status error", err)
	}
	if result.StatusCode != http.StatusNotFound {
		t.Fatalf("result status = %d", result.StatusCode)
	}
}
