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
	defaultRequestTimeout = 15 * time.Second
	defaultMaxBodyBytes   = 8 << 20
	defaultMaxRedirects   = 5
	defaultMaxAttempts    = 3
	defaultBaseBackoff    = 250 * time.Millisecond
	defaultMaxBackoff     = 2 * time.Second
	defaultUserAgent      = "GoreeCloud-Feeds/0.1.0-dev"
)

var (
	ErrInvalidURL         = errors.New("invalid feed URL")
	ErrBlockedDestination = errors.New("feed destination is blocked by retrieval policy")
	ErrTooManyRedirects   = errors.New("feed retrieval exceeded redirect limit")
	ErrDowngradeRedirect  = errors.New("feed retrieval blocked HTTPS-to-HTTP redirect")
	ErrBodyTooLarge       = errors.New("feed response exceeds configured body limit")
)

// HTTPStatusError reports a non-success feed response after retry handling.
type HTTPStatusError struct {
	StatusCode int
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("feed retrieval returned HTTP %d", e.StatusCode)
}

// Resolver is the minimum DNS boundary required by the retrieval client.
type Resolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

// Conditions carries validators from a previous successful retrieval.
type Conditions struct {
	ETag         string
	LastModified string
}

// Result is the bounded response from one successful or not-modified retrieval.
type Result struct {
	StatusCode   int
	FinalURL     string
	ContentType  string
	Body         []byte
	ETag         string
	LastModified string
	NotModified  bool
}

// Config controls bounded retrieval behavior. Zero values select conservative
// Development defaults. AllowCIDRs may explicitly permit otherwise blocked
// address ranges for separately approved private-network feeds.
type Config struct {
	RequestTimeout time.Duration
	MaxBodyBytes   int64
	MaxRedirects   int
	MaxAttempts    int
	BaseBackoff    time.Duration
	MaxBackoff     time.Duration
	AllowCIDRs     []netip.Prefix
	Resolver       Resolver
	Dialer         *net.Dialer
	Sleep          func(context.Context, time.Duration) error
	UserAgent      string
}

// Fetcher retrieves feed documents without exposing an HTTP handler or
// persistence behavior. Callers remain responsible for scheduling, history,
// parsing, deduplication, persistence, and authorization.
type Fetcher struct {
	config   Config
	client   *http.Client
	resolver Resolver
	dialer   *net.Dialer
	sleep    func(context.Context, time.Duration) error
}

var specialUsePrefixes = mustPrefixes(
	"0.0.0.0/8",
	"100.64.0.0/10",
	"192.0.0.0/24",
	"192.0.2.0/24",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"2001:db8::/32",
)

func mustPrefixes(values ...string) []netip.Prefix {
	out := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		out = append(out, netip.MustParsePrefix(value))
	}
	return out
}

// New constructs a retrieval client with environment proxies disabled, TLS
// verification enabled, a TLS 1.2 minimum, bounded redirects, and DNS/IP
// destination validation before every connection.
func New(config Config) (*Fetcher, error) {
	if config.RequestTimeout < 0 || config.MaxBodyBytes < 0 || config.MaxRedirects < 0 ||
		config.MaxAttempts < 0 || config.BaseBackoff < 0 || config.MaxBackoff < 0 {
		return nil, errors.New("retrieval limits must not be negative")
	}
	if config.RequestTimeout == 0 {
		config.RequestTimeout = defaultRequestTimeout
	}
	if config.MaxBodyBytes == 0 {
		config.MaxBodyBytes = defaultMaxBodyBytes
	}
	if config.MaxRedirects == 0 {
		config.MaxRedirects = defaultMaxRedirects
	}
	if config.MaxAttempts == 0 {
		config.MaxAttempts = defaultMaxAttempts
	}
	if config.BaseBackoff == 0 {
		config.BaseBackoff = defaultBaseBackoff
	}
	if config.MaxBackoff == 0 {
		config.MaxBackoff = defaultMaxBackoff
	}
	if config.MaxBackoff < config.BaseBackoff {
		return nil, errors.New("maximum backoff must be greater than or equal to base backoff")
	}
	if config.UserAgent == "" {
		config.UserAgent = defaultUserAgent
	}
	if strings.TrimSpace(config.UserAgent) == "" {
		return nil, errors.New("user agent must not be blank")
	}
	for _, prefix := range config.AllowCIDRs {
		if !prefix.IsValid() {
			return nil, errors.New("retrieval allow CIDR is invalid")
		}
	}

	resolver := config.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	dialer := config.Dialer
	if dialer == nil {
		dialer = &net.Dialer{
			Timeout:   config.RequestTimeout,
			KeepAlive: 30 * time.Second,
		}
	}
	sleep := config.Sleep
	if sleep == nil {
		sleep = sleepContext
	}

	fetcher := &Fetcher{
		config:   config,
		resolver: resolver,
		dialer:   dialer,
		sleep:    sleep,
	}

	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           fetcher.dialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          32,
		MaxIdleConnsPerHost:   4,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   minDuration(config.RequestTimeout, 10*time.Second),
		ResponseHeaderTimeout: config.RequestTimeout,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	fetcher.client = &http.Client{
		Transport:     transport,
		CheckRedirect: fetcher.checkRedirect,
	}
	return fetcher, nil
}

// Fetch retrieves one feed document using bounded retries and conditional
// request validators. It returns an HTTPStatusError for final non-success
// responses while preserving the bounded result metadata.
func (f *Fetcher) Fetch(ctx context.Context, rawURL string, conditions Conditions) (Result, error) {
	target, err := parseAbsoluteURL(rawURL)
	if err != nil {
		return Result{}, err
	}
	if err := f.validateTarget(ctx, target); err != nil {
		return Result{}, err
	}

	var last Result
	for attempt := 1; attempt <= f.config.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}

		result, err := f.fetchOnce(ctx, target, conditions)
		last = result
		if err == nil {
			if retryableStatus(result.StatusCode) && attempt < f.config.MaxAttempts {
				if err := f.sleep(ctx, f.backoff(attempt)); err != nil {
					return Result{}, err
				}
				continue
			}
			if result.StatusCode >= 200 && result.StatusCode < 300 || result.NotModified {
				return result, nil
			}
			return result, &HTTPStatusError{StatusCode: result.StatusCode}
		}

		if !retryableError(ctx, err) || attempt == f.config.MaxAttempts {
			return result, err
		}
		if err := f.sleep(ctx, f.backoff(attempt)); err != nil {
			return Result{}, err
		}
	}
	return last, errors.New("feed retrieval exhausted attempts")
}

func (f *Fetcher) fetchOnce(parent context.Context, target *url.URL, conditions Conditions) (Result, error) {
	ctx, cancel := context.WithTimeout(parent, f.config.RequestTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}
	request.Header.Set("Accept", "application/atom+xml, application/rss+xml, application/xml, text/xml;q=0.9, */*;q=0.1")
	request.Header.Set("User-Agent", f.config.UserAgent)
	request.Header.Set("Cache-Control", "no-cache")
	request.Header.Set("Pragma", "no-cache")
	if value := strings.TrimSpace(conditions.ETag); value != "" {
		request.Header.Set("If-None-Match", value)
	}
	if value := strings.TrimSpace(conditions.LastModified); value != "" {
		request.Header.Set("If-Modified-Since", value)
	}

	response, err := f.client.Do(request)
	if err != nil {
		return Result{}, err
	}
	defer response.Body.Close()

	result := Result{
		StatusCode:   response.StatusCode,
		FinalURL:     response.Request.URL.String(),
		ContentType:  response.Header.Get("Content-Type"),
		ETag:         response.Header.Get("ETag"),
		LastModified: response.Header.Get("Last-Modified"),
		NotModified:  response.StatusCode == http.StatusNotModified,
	}
	if result.NotModified {
		return result, nil
	}

	if response.ContentLength > f.config.MaxBodyBytes {
		return result, ErrBodyTooLarge
	}
	limited := io.LimitReader(response.Body, f.config.MaxBodyBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return result, fmt.Errorf("read feed response: %w", err)
	}
	if int64(len(body)) > f.config.MaxBodyBytes {
		return result, ErrBodyTooLarge
	}
	result.Body = body
	return result, nil
}

func (f *Fetcher) checkRedirect(request *http.Request, via []*http.Request) error {
	if len(via) > f.config.MaxRedirects {
		return ErrTooManyRedirects
	}
	if len(via) > 0 && strings.EqualFold(via[len(via)-1].URL.Scheme, "https") &&
		strings.EqualFold(request.URL.Scheme, "http") {
		return ErrDowngradeRedirect
	}
	if err := f.validateTarget(request.Context(), request.URL); err != nil {
		return err
	}

	request.Header.Del("Authorization")
	request.Header.Del("Cookie")
	request.Header.Del("Proxy-Authorization")
	request.Header.Del("Referer")
	return nil
}

func (f *Fetcher) dialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid network address", ErrInvalidURL)
	}
	addresses, err := f.resolveAndValidate(ctx, host)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for _, candidate := range addresses {
		conn, err := f.dialer.DialContext(ctx, network, net.JoinHostPort(candidate.String(), port))
		if err != nil {
			lastErr = err
			continue
		}
		if err := f.validateRemoteAddr(conn.RemoteAddr()); err != nil {
			_ = conn.Close()
			return nil, err
		}
		return conn, nil
	}
	if lastErr == nil {
		lastErr = errors.New("no validated destination addresses")
	}
	return nil, lastErr
}

func (f *Fetcher) validateTarget(ctx context.Context, target *url.URL) error {
	if target == nil || !target.IsAbs() || target.Hostname() == "" {
		return ErrInvalidURL
	}
	if target.Scheme != "http" && target.Scheme != "https" {
		return fmt.Errorf("%w: only HTTP and HTTPS are supported", ErrInvalidURL)
	}
	if target.User != nil {
		return fmt.Errorf("%w: embedded URL credentials are not allowed", ErrInvalidURL)
	}
	if port := target.Port(); port != "" {
		value, err := strconv.Atoi(port)
		if err != nil || value < 1 || value > 65535 {
			return fmt.Errorf("%w: invalid port", ErrInvalidURL)
		}
	}
	_, err := f.resolveAndValidate(ctx, target.Hostname())
	return err
}

func (f *Fetcher) resolveAndValidate(ctx context.Context, host string) ([]netip.Addr, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return nil, ErrInvalidURL
	}
	if strings.Contains(host, "%") {
		return nil, fmt.Errorf("%w: scoped IP literals are not allowed", ErrInvalidURL)
	}

	var addresses []netip.Addr
	if literal, err := netip.ParseAddr(host); err == nil {
		addresses = []netip.Addr{literal.Unmap()}
	} else {
		resolved, err := f.resolver.LookupNetIP(ctx, "ip", host)
		if err != nil {
			return nil, fmt.Errorf("resolve feed destination: %w", err)
		}
		for _, addr := range resolved {
			if addr.IsValid() {
				addresses = append(addresses, addr.Unmap())
			}
		}
	}
	if len(addresses) == 0 {
		return nil, errors.New("feed destination resolved to no addresses")
	}

	seen := make(map[netip.Addr]struct{}, len(addresses))
	validated := make([]netip.Addr, 0, len(addresses))
	for _, addr := range addresses {
		if _, ok := seen[addr]; ok {
			continue
		}
		seen[addr] = struct{}{}
		if !f.addressAllowed(addr) {
			return nil, fmt.Errorf("%w: %s", ErrBlockedDestination, addr)
		}
		validated = append(validated, addr)
	}
	return validated, nil
}

func (f *Fetcher) validateRemoteAddr(remote net.Addr) error {
	switch value := remote.(type) {
	case *net.TCPAddr:
		addr, ok := netip.AddrFromSlice(value.IP)
		if !ok || !f.addressAllowed(addr.Unmap()) {
			return ErrBlockedDestination
		}
		return nil
	default:
		host, _, err := net.SplitHostPort(remote.String())
		if err != nil {
			return ErrBlockedDestination
		}
		addr, err := netip.ParseAddr(host)
		if err != nil || !f.addressAllowed(addr.Unmap()) {
			return ErrBlockedDestination
		}
		return nil
	}
}

func (f *Fetcher) addressAllowed(addr netip.Addr) bool {
	if !addr.IsValid() {
		return false
	}
	addr = addr.Unmap()
	for _, prefix := range f.config.AllowCIDRs {
		if prefix.Contains(addr) {
			return true
		}
	}
	if !addr.IsGlobalUnicast() || addr.IsPrivate() || addr.IsLoopback() ||
		addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsMulticast() ||
		addr.IsUnspecified() {
		return false
	}
	for _, prefix := range specialUsePrefixes {
		if prefix.Contains(addr) {
			return false
		}
	}
	return true
}

func (f *Fetcher) backoff(attempt int) time.Duration {
	delay := f.config.BaseBackoff
	for i := 1; i < attempt && delay < f.config.MaxBackoff; i++ {
		if delay > f.config.MaxBackoff/2 {
			return f.config.MaxBackoff
		}
		delay *= 2
	}
	if delay > f.config.MaxBackoff {
		return f.config.MaxBackoff
	}
	return delay
}

func parseAbsoluteURL(raw string) (*url.URL, error) {
	target, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || target == nil || !target.IsAbs() {
		return nil, fmt.Errorf("%w: absolute URL required", ErrInvalidURL)
	}
	return target, nil
}

func retryableStatus(status int) bool {
	switch status {
	case http.StatusRequestTimeout, http.StatusTooEarly, http.StatusTooManyRequests,
		http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func retryableError(parent context.Context, err error) bool {
	if err == nil || parent.Err() != nil {
		return false
	}
	return !errors.Is(err, ErrInvalidURL) &&
		!errors.Is(err, ErrBlockedDestination) &&
		!errors.Is(err, ErrTooManyRedirects) &&
		!errors.Is(err, ErrDowngradeRedirect) &&
		!errors.Is(err, ErrBodyTooLarge)
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func minDuration(left, right time.Duration) time.Duration {
	if left < right {
		return left
	}
	return right
}
