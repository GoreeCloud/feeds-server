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

func TestCrossOriginRedirectDropsOriginSpecificHeaders(t *testing.T) {
	client := loopbackClient(t, 1024, 2)
	previous := httptest.NewRequest(http.MethodGet, "https://one.example/feed.xml", nil)
	next := httptest.NewRequest(http.MethodGet, "https://two.example/feed.xml", nil)
	next.Header.Set("If-None-Match", `"private-validator"`)
	next.Header.Set("If-Modified-Since", "yesterday")
	next.Header.Set("Authorization", "Bearer should-not-forward")
	next.Header.Set("Proxy-Authorization", "Basic should-not-forward")
	next.Header.Set("Cookie", "session=should-not-forward")
	next.Header.Set("Referer", "https://one.example/private")
	if err := client.checkRedirect(next, []*http.Request{previous}); err != nil {
		t.Fatalf("checkRedirect() error = %v", err)
	}
	for _, name := range []string{
		"If-None-Match",
		"If-Modified-Since",
		"Authorization",
		"Proxy-Authorization",
		"Cookie",
		"Referer",
	} {
		if got := next.Header.Get(name); got != "" {
			t.Fatalf("%s retained across redirect: %q", name, got)
		}
	}
}

func TestPortChangeIsCrossOriginForValidators(t *testing.T) {
	client := loopbackClient(t, 1024, 2)
	previous := httptest.NewRequest(http.MethodGet, "https://example.com:443/feed.xml", nil)
	next := httptest.NewRequest(http.MethodGet, "https://example.com:8443/feed.xml", nil)
	next.Header.Set("If-None-Match", `"origin-validator"`)
	if err := client.checkRedirect(next, []*http.Request{previous}); err != nil {
		t.Fatalf("checkRedirect() error = %v", err)
	}
	if got := next.Header.Get("If-None-Match"); got != "" {
		t.Fatalf("validator retained across port-changing redirect: %q", got)
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


func TestRetryableStatusUsesBoundedBackoff(t *testing.T) {
	var requests int32
	var delays []time.Duration
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requests, 1)
		if count < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		RequestTimeout:  2 * time.Second,
		ConnectTimeout:  time.Second,
		MaxBodyBytes:    1024,
		MaxConcurrent:   1,
		MaxAttempts:     3,
		BaseBackoff:     10 * time.Millisecond,
		MaxBackoff:      20 * time.Millisecond,
		AllowPlainHTTP:  true,
		AllowedNetworks: []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8")},
		Sleep: func(_ context.Context, delay time.Duration) error {
			delays = append(delays, delay)
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := client.Fetch(context.Background(), FetchRequest{URL: server.URL})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if requests != 3 || result.Attempts != 3 {
		t.Fatalf("requests=%d attempts=%d", requests, result.Attempts)
	}
	if len(delays) != 2 || delays[0] != 10*time.Millisecond || delays[1] != 20*time.Millisecond {
		t.Fatalf("retry delays = %v", delays)
	}
}

func TestRetryAfterIsHonoredButBounded(t *testing.T) {
	var requests int32
	var delays []time.Duration
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&requests, 1) == 1 {
			w.Header().Set("Retry-After", "120")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		RequestTimeout:  2 * time.Second,
		ConnectTimeout:  time.Second,
		MaxBodyBytes:    1024,
		MaxConcurrent:   1,
		MaxAttempts:     2,
		BaseBackoff:     5 * time.Millisecond,
		MaxBackoff:      25 * time.Millisecond,
		AllowPlainHTTP:  true,
		AllowedNetworks: []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8")},
		Sleep: func(_ context.Context, delay time.Duration) error {
			delays = append(delays, delay)
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := client.Fetch(context.Background(), FetchRequest{URL: server.URL})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if result.Attempts != 2 || len(delays) != 1 || delays[0] != 25*time.Millisecond {
		t.Fatalf("attempts=%d delays=%v", result.Attempts, delays)
	}
}

func TestPermanentHTTPFailureIsNotRetried(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		RequestTimeout:  2 * time.Second,
		ConnectTimeout:  time.Second,
		MaxBodyBytes:    1024,
		MaxConcurrent:   1,
		MaxAttempts:     3,
		AllowPlainHTTP:  true,
		AllowedNetworks: []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8")},
		Sleep: func(context.Context, time.Duration) error {
			t.Fatal("sleep called for permanent failure")
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := client.Fetch(context.Background(), FetchRequest{URL: server.URL})
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusNotFound {
		t.Fatalf("Fetch() error = %v", err)
	}
	if requests != 1 || result.Attempts != 1 {
		t.Fatalf("requests=%d attempts=%d", requests, result.Attempts)
	}
}

func TestCancelledContextStopsRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var sleeps int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		RequestTimeout:  2 * time.Second,
		ConnectTimeout:  time.Second,
		MaxBodyBytes:    1024,
		MaxConcurrent:   1,
		MaxAttempts:     3,
		AllowPlainHTTP:  true,
		AllowedNetworks: []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8")},
		Sleep: func(context.Context, time.Duration) error {
			atomic.AddInt32(&sleeps, 1)
			cancel()
			return context.Canceled
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Fetch(ctx, FetchRequest{URL: server.URL})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Fetch() error = %v, want context canceled", err)
	}
	if sleeps != 1 {
		t.Fatalf("sleep calls = %d, want 1", sleeps)
	}
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 9, 19, 20, 0, 0, 0, time.UTC)
	if got := parseRetryAfter("3", now); got != 3*time.Second {
		t.Fatalf("seconds Retry-After = %v", got)
	}
	if got := parseRetryAfter("Sat, 19 Sep 2026 20:00:05 GMT", now); got != 5*time.Second {
		t.Fatalf("date Retry-After = %v", got)
	}
	if got := parseRetryAfter("not-valid", now); got != -1 {
		t.Fatalf("invalid Retry-After = %v", got)
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
