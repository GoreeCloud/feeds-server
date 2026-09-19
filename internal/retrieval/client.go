package retrieval

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultRequestTimeout              = 30 * time.Second
	defaultConnectTimeout              = 5 * time.Second
	defaultResponseHeaderTimeout       = 10 * time.Second
	defaultMaxBodyBytes          int64 = 8 << 20
	defaultMaxConcurrent               = 8
	defaultMaxAttempts                 = 3
	defaultBaseBackoff                 = 250 * time.Millisecond
	defaultMaxBackoff                  = 2 * time.Second
	defaultUserAgent                   = "GoreeCloud-Feeds/0.1.0-dev"
	maxRedirects                       = 5
)

var (
	ErrResponseTooLarge = errors.New("feed retrieval response exceeds configured size limit")
	ErrHTTPSDowngrade   = errors.New("feed retrieval redirect from HTTPS to HTTP is blocked")
	ErrTooManyRedirects = errors.New("feed retrieval exceeded redirect limit")
)

// Config defines safe defaults and bounded resource use for remote feed
// retrieval. Private destinations remain denied unless explicitly allowlisted.
type Config struct {
	RequestTimeout        time.Duration
	ConnectTimeout        time.Duration
	ResponseHeaderTimeout time.Duration
	MaxBodyBytes          int64
	MaxConcurrent         int
	MaxAttempts           int
	BaseBackoff           time.Duration
	MaxBackoff            time.Duration
	AllowPlainHTTP        bool
	AllowedNetworks       []netip.Prefix
	UserAgent             string
	Resolver              Resolver
	Sleep                 func(context.Context, time.Duration) error
}

type Client struct {
	httpClient   *http.Client
	policy       DestinationPolicy
	resolver     Resolver
	maxBodyBytes int64
	userAgent    string
	semaphore    chan struct{}
	maxAttempts  int
	baseBackoff  time.Duration
	maxBackoff   time.Duration
	sleep        func(context.Context, time.Duration) error
}

type FetchRequest struct {
	URL          string
	ETag         string
	LastModified string
}

type FetchResult struct {
	URL          string
	StatusCode   int
	ETag         string
	LastModified string
	ContentType  string
	Body         []byte
	NotModified  bool
	FetchedAt    time.Time
	Duration     time.Duration
	Attempts     int
}

type HTTPStatusError struct {
	StatusCode int
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("feed retrieval returned HTTP %d", e.StatusCode)
}

func NewClient(cfg Config) (*Client, error) {
	if cfg.RequestTimeout < 0 || cfg.ConnectTimeout < 0 || cfg.ResponseHeaderTimeout < 0 ||
		cfg.MaxBodyBytes < 0 || cfg.MaxConcurrent < 0 || cfg.MaxAttempts < 0 ||
		cfg.BaseBackoff < 0 || cfg.MaxBackoff < 0 {
		return nil, errors.New("feed retrieval configuration values must not be negative")
	}
	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = defaultRequestTimeout
	}
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = defaultConnectTimeout
	}
	if cfg.ResponseHeaderTimeout == 0 {
		cfg.ResponseHeaderTimeout = defaultResponseHeaderTimeout
	}
	if cfg.MaxBodyBytes == 0 {
		cfg.MaxBodyBytes = defaultMaxBodyBytes
	}
	if cfg.MaxConcurrent == 0 {
		cfg.MaxConcurrent = defaultMaxConcurrent
	}
	if cfg.MaxAttempts == 0 {
		cfg.MaxAttempts = defaultMaxAttempts
	}
	if cfg.BaseBackoff == 0 {
		cfg.BaseBackoff = defaultBaseBackoff
	}
	if cfg.MaxBackoff == 0 {
		cfg.MaxBackoff = defaultMaxBackoff
	}
	if cfg.MaxBackoff < cfg.BaseBackoff {
		return nil, errors.New("feed retrieval maximum backoff must be greater than or equal to base backoff")
	}
	if strings.TrimSpace(cfg.UserAgent) == "" {
		cfg.UserAgent = defaultUserAgent
	}
	if cfg.Resolver == nil {
		cfg.Resolver = net.DefaultResolver
	}
	if cfg.Sleep == nil {
		cfg.Sleep = sleepContext
	}

	client := &Client{
		policy: DestinationPolicy{
			AllowPlainHTTP:  cfg.AllowPlainHTTP,
			AllowedNetworks: append([]netip.Prefix(nil), cfg.AllowedNetworks...),
		},
		resolver:     cfg.Resolver,
		maxBodyBytes: cfg.MaxBodyBytes,
		userAgent:    cfg.UserAgent,
		semaphore:    make(chan struct{}, cfg.MaxConcurrent),
		maxAttempts:  cfg.MaxAttempts,
		baseBackoff:  cfg.BaseBackoff,
		maxBackoff:   cfg.MaxBackoff,
		sleep:        cfg.Sleep,
	}

	dialer := &net.Dialer{
		Timeout:   cfg.ConnectTimeout,
		KeepAlive: 30 * time.Second,
	}
	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           client.dialContext(dialer),
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          64,
		MaxIdleConnsPerHost:   8,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: cfg.ResponseHeaderTimeout,
		ExpectContinueTimeout: time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}
	client.httpClient = &http.Client{
		Transport:     transport,
		Timeout:       cfg.RequestTimeout,
		CheckRedirect: client.checkRedirect,
	}
	return client, nil
}

func (c *Client) Fetch(ctx context.Context, input FetchRequest) (FetchResult, error) {
	select {
	case c.semaphore <- struct{}{}:
		defer func() { <-c.semaphore }()
	case <-ctx.Done():
		return FetchResult{}, ctx.Err()
	}

	rawURL := strings.TrimSpace(input.URL)
	u, err := url.Parse(rawURL)
	if err != nil {
		return FetchResult{}, fmt.Errorf("parse feed retrieval URL: %w", err)
	}
	if err := c.policy.ValidateURL(u); err != nil {
		return FetchResult{}, err
	}

	started := time.Now()
	var last FetchResult
	for attempt := 1; attempt <= c.maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return last, err
		}

		result, retryAfter, err := c.fetchAttempt(ctx, u, input, started, attempt)
		last = result
		if err == nil {
			return result, nil
		}
		if attempt == c.maxAttempts || !c.shouldRetry(ctx, err) {
			return result, err
		}

		delay := c.backoff(attempt)
		if retryAfter >= 0 {
			delay = retryAfter
		}
		if err := c.sleep(ctx, delay); err != nil {
			return result, err
		}
	}
	return last, errors.New("feed retrieval exhausted configured attempts")
}

func (c *Client) fetchAttempt(
	ctx context.Context,
	u *url.URL,
	input FetchRequest,
	started time.Time,
	attempt int,
) (FetchResult, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return FetchResult{Attempts: attempt, Duration: time.Since(started)}, -1, fmt.Errorf("create feed retrieval request: %w", err)
	}
	req.Header.Set("Accept", "application/atom+xml, application/rss+xml, application/xml, text/xml, */*;q=0.1")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	if etag := strings.TrimSpace(input.ETag); etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	if modified := strings.TrimSpace(input.LastModified); modified != "" {
		req.Header.Set("If-Modified-Since", modified)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return FetchResult{Attempts: attempt, Duration: time.Since(started)}, -1, fmt.Errorf("retrieve feed: %w", err)
	}
	defer resp.Body.Close()

	result := FetchResult{
		URL:          resp.Request.URL.String(),
		StatusCode:   resp.StatusCode,
		ETag:         strings.TrimSpace(resp.Header.Get("ETag")),
		LastModified: strings.TrimSpace(resp.Header.Get("Last-Modified")),
		ContentType:  strings.TrimSpace(resp.Header.Get("Content-Type")),
		FetchedAt:    time.Now().UTC(),
		Attempts:     attempt,
	}
	finish := func() FetchResult {
		result.Duration = time.Since(started)
		return result
	}

	if resp.StatusCode == http.StatusNotModified {
		result.NotModified = true
		return finish(), -1, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return finish(), parseRetryAfter(resp.Header.Get("Retry-After"), time.Now(), c.maxBackoff), &HTTPStatusError{StatusCode: resp.StatusCode}
	}
	if resp.ContentLength > c.maxBodyBytes {
		return finish(), -1, ErrResponseTooLarge
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, c.maxBodyBytes+1))
	if err != nil {
		return finish(), -1, fmt.Errorf("read feed retrieval response: %w", err)
	}
	if int64(len(body)) > c.maxBodyBytes {
		return finish(), -1, ErrResponseTooLarge
	}
	result.Body = body
	return finish(), -1, nil
}

func (c *Client) shouldRetry(ctx context.Context, err error) bool {
	if err == nil || ctx.Err() != nil {
		return false
	}

	var statusErr *HTTPStatusError
	if errors.As(err, &statusErr) {
		switch statusErr.StatusCode {
		case http.StatusRequestTimeout,
			http.StatusTooEarly,
			http.StatusTooManyRequests,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			return true
		default:
			return false
		}
	}

	return !errors.Is(err, ErrResponseTooLarge) &&
		!errors.Is(err, ErrHTTPSDowngrade) &&
		!errors.Is(err, ErrTooManyRedirects) &&
		!errors.Is(err, ErrDestinationBlocked) &&
		!errors.Is(err, ErrEmbeddedCredentials) &&
		!errors.Is(err, ErrUnsupportedScheme)
}

func (c *Client) backoff(attempt int) time.Duration {
	delay := c.baseBackoff
	for step := 1; step < attempt && delay < c.maxBackoff; step++ {
		if delay > c.maxBackoff/2 {
			return c.maxBackoff
		}
		delay *= 2
	}
	if delay > c.maxBackoff {
		return c.maxBackoff
	}
	return delay
}

func (c *Client) checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) > maxRedirects {
		return ErrTooManyRedirects
	}
	if len(via) > 0 && strings.EqualFold(via[len(via)-1].URL.Scheme, "https") &&
		strings.EqualFold(req.URL.Scheme, "http") {
		return ErrHTTPSDowngrade
	}
	if err := c.policy.ValidateURL(req.URL); err != nil {
		return err
	}

	if len(via) > 0 && !sameOrigin(via[len(via)-1].URL, req.URL) {
		req.Header.Del("If-None-Match")
		req.Header.Del("If-Modified-Since")
	}
	req.Header.Del("Authorization")
	req.Header.Del("Proxy-Authorization")
	req.Header.Del("Cookie")
	req.Header.Del("Referer")
	return nil
}

func sameOrigin(left, right *url.URL) bool {
	if left == nil || right == nil {
		return false
	}
	return strings.EqualFold(left.Scheme, right.Scheme) &&
		strings.EqualFold(left.Hostname(), right.Hostname()) &&
		effectivePort(left) == effectivePort(right)
}

func effectivePort(u *url.URL) string {
	if u == nil {
		return ""
	}
	if port := u.Port(); port != "" {
		return port
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return "443"
	case "http":
		return "80"
	default:
		return ""
	}
}

func (c *Client) dialContext(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("parse retrieval destination %q: %w", address, err)
		}
		addrs, err := c.policy.resolveAllowed(ctx, c.resolver, host)
		if err != nil {
			return nil, err
		}

		var dialErrors []error
		for _, addr := range addrs {
			if network == "tcp4" && !addr.Is4() {
				continue
			}
			if network == "tcp6" && !addr.Is6() {
				continue
			}
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(addr.String(), port))
			if err == nil {
				return conn, nil
			}
			dialErrors = append(dialErrors, err)
		}
		if len(dialErrors) == 0 {
			return nil, fmt.Errorf("no resolved address matched network %q for %q", network, host)
		}
		return nil, fmt.Errorf("dial retrieval destination %q: %w", host, errors.Join(dialErrors...))
	}
}

func parseRetryAfter(raw string, now time.Time, maximum time.Duration) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return -1
	}
	if seconds, err := strconv.ParseInt(raw, 10, 64); err == nil {
		if seconds < 0 {
			return -1
		}
		if maximum <= 0 {
			return 0
		}
		maxSeconds := maximum / time.Second
		if maximum%time.Second != 0 {
			maxSeconds++
		}
		if seconds > int64(maxSeconds) {
			return maximum
		}
		delay := time.Duration(seconds) * time.Second
		if delay > maximum {
			return maximum
		}
		return delay
	}
	if when, err := http.ParseTime(raw); err == nil {
		delay := when.Sub(now)
		if delay < 0 {
			return 0
		}
		if delay > maximum {
			return maximum
		}
		return delay
	}
	return -1
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
