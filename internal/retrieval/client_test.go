package retrieval

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fakeResolver struct {
	addrs []netip.Addr
	err   error
}

func (r fakeResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return append([]netip.Addr(nil), r.addrs...), r.err
}

func loopbackClient(t *testing.T, maxBody int64, maxConcurrent int) *Client {
	t.Helper()
	client, err := NewClient(Config{
		RequestTimeout:  2 * time.Second,
		ConnectTimeout:  time.Second,
		MaxBodyBytes:    maxBody,
		MaxConcurrent:   maxConcurrent,
		AllowPlainHTTP:  true,
		AllowedNetworks: []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8"), netip.MustParsePrefix("::1/128")},
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return client
}

func TestDestinationPolicyRejectsUnsafeURLs(t *testing.T) {
	policy := DestinationPolicy{}
	for _, raw := range []string{
		"ftp://example.com/feed.xml",
		"http://example.com/feed.xml",
		"https://user:pass@example.com/feed.xml",
		"https://127.0.0.1/feed.xml",
		"https://10.0.0.1/feed.xml",
		"https://169.254.169.254/latest/meta-data/",
	} {
		u := mustURL(t, raw)
		if err := policy.ValidateURL(u); err == nil {
			t.Fatalf("ValidateURL(%q) unexpectedly succeeded", raw)
		}
	}
}

func TestDestinationPolicyRejectsMixedSafeAndBlockedDNSAnswers(t *testing.T) {
	policy := DestinationPolicy{}
	_, err := policy.resolveAllowed(context.Background(), fakeResolver{addrs: []netip.Addr{
		netip.MustParseAddr("93.184.216.34"),
		netip.MustParseAddr("127.0.0.1"),
	}}, "example.test")
	if !errors.Is(err, ErrDestinationBlocked) {
		t.Fatalf("expected ErrDestinationBlocked, got %v", err)
	}
}

func TestFetchReturnsBoundedFeedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s", r.Method)
		}
		if got := r.Header.Get("User-Agent"); got != defaultUserAgent {
			t.Fatalf("user agent = %q", got)
		}
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Header().Set("ETag", `"feed-v1"`)
		_, _ = w.Write([]byte("<rss><channel><title>Example</title></channel></rss>"))
	}))
	defer server.Close()

	client := loopbackClient(t, 1024, 2)
	result, err := client.Fetch(context.Background(), FetchRequest{URL: server.URL + "/feed.xml"})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if result.StatusCode != http.StatusOK || result.ETag != `"feed-v1"` || result.NotModified {
		t.Fatalf("unexpected result: %+v", result)
	}
	if string(result.Body) != "<rss><channel><title>Example</title></channel></rss>" {
		t.Fatalf("body = %q", result.Body)
	}
}

func TestFetchUsesConditionalHeadersAndHandlesNotModified(t *testing.T) {
	const modified = "Fri, 18 Sep 2026 12:30:00 GMT"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("If-None-Match"); got != `"feed-v1"` {
			t.Fatalf("If-None-Match = %q", got)
		}
		if got := r.Header.Get("If-Modified-Since"); got != modified {
			t.Fatalf("If-Modified-Since = %q", got)
		}
		w.Header().Set("ETag", `"feed-v1"`)
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	client := loopbackClient(t, 1024, 2)
	result, err := client.Fetch(context.Background(), FetchRequest{
		URL:          server.URL + "/feed.xml",
		ETag:         `"feed-v1"`,
		LastModified: modified,
	})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if !result.NotModified || len(result.Body) != 0 || result.StatusCode != http.StatusNotModified {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestFetchRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("0123456789abcdef"))
	}))
	defer server.Close()

	client := loopbackClient(t, 8, 2)
	_, err := client.Fetch(context.Background(), FetchRequest{URL: server.URL})
	if !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("expected ErrResponseTooLarge, got %v", err)
	}
}

func TestRedirectToBlockedDestinationIsRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://169.254.169.254/latest/meta-data/", http.StatusFound)
	}))
	defer server.Close()

	client := loopbackClient(t, 1024, 2)
	_, err := client.Fetch(context.Background(), FetchRequest{URL: server.URL})
	if !errors.Is(err, ErrDestinationBlocked) {
		t.Fatalf("expected ErrDestinationBlocked, got %v", err)
	}
}

func TestHTTPSDowngradeRedirectIsRejected(t *testing.T) {
	client := loopbackClient(t, 1024, 2)
	previous := httptest.NewRequest(http.MethodGet, "https://example.com/feed.xml", nil)
	next := httptest.NewRequest(http.MethodGet, "http://example.com/feed.xml", nil)
	if err := client.checkRedirect(next, []*http.Request{previous}); !errors.Is(err, ErrHTTPSDowngrade) {
		t.Fatalf("expected ErrHTTPSDowngrade, got %v", err)
	}
}

func TestCrossHostRedirectDropsConditionalValidators(t *testing.T) {
	client := loopbackClient(t, 1024, 2)
	previous := httptest.NewRequest(http.MethodGet, "https://one.example/feed.xml", nil)
	next := httptest.NewRequest(http.MethodGet, "https://two.example/feed.xml", nil)
	next.Header.Set("If-None-Match", `"private-validator"`)
	next.Header.Set("If-Modified-Since", "yesterday")
	if err := client.checkRedirect(next, []*http.Request{previous}); err != nil {
		t.Fatalf("checkRedirect() error = %v", err)
	}
	if next.Header.Get("If-None-Match") != "" || next.Header.Get("If-Modified-Since") != "" {
		t.Fatal("cross-host redirect retained conditional validators")
	}
}

func TestConcurrencyIsBounded(t *testing.T) {
	var active int32
	var peak int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&active, 1)
		defer atomic.AddInt32(&active, -1)
		for {
			old := atomic.LoadInt32(&peak)
			if current <= old || atomic.CompareAndSwapInt32(&peak, old, current) {
				break
			}
		}
		time.Sleep(40 * time.Millisecond)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := loopbackClient(t, 1024, 2)
	var wg sync.WaitGroup
	errCh := make(chan error, 6)
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := client.Fetch(context.Background(), FetchRequest{URL: server.URL})
			errCh <- err
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("Fetch() error = %v", err)
		}
	}
	if got := atomic.LoadInt32(&peak); got > 2 {
		t.Fatalf("peak concurrency = %d, want <= 2", got)
	}
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse(%q) error = %v", raw, err)
	}
	return u
}
