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
	"strings"
	"time"
)

const (
	defaultRequestTimeout              = 30 * time.Second
	defaultConnectTimeout              = 5 * time.Second
	defaultResponseHeaderTimeout       = 10 * time.Second
	defaultMaxBodyBytes          int64 = 8 << 20
	defaultMaxConcurrent               = 8
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
	AllowPlainHTTP        bool
	AllowedNetworks       []netip.Prefix
	UserAgent             string
	Resolver              Resolver
}

type Client struct {
	httpClient   *http.Client
	policy       DestinationPolicy
	resolver     Resolver
	maxBodyBytes int64
	userAgent    string
	semaphore    chan struct{}
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
}

type HTTPStatusError struct {
	StatusCode int
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("feed retrieval returned HTTP %d", e.StatusCode)
}

func NewClient(cfg Config) (*Client, error) {
	if cfg.RequestTimeout < 0 || cfg.ConnectTimeout < 0 || cfg.ResponseHeaderTimeout < 0 || cfg.MaxBodyBytes < 0 || cfg.MaxConcurrent < 0 {
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
	if strings.TrimSpace(cfg.UserAgent) == "" {
		cfg.UserAgent = defaultUserAgent
	}
	if cfg.Resolver == nil {
		cfg.Resolver = net.DefaultResolver
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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return FetchResult{}, fmt.Errorf("create feed retrieval request: %w", err)
	}
	req.Header.Set("Accept", "application/atom+xml, application/rss+xml, application/xml, text/xml, */*;q=0.1")
	req.Header.Set("User-Agent", c.userAgent)
	if etag := strings.TrimSpace(input.ETag); etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	if modified := strings.TrimSpace(input.LastModified); modified != "" {
		req.Header.Set("If-Modified-Since", modified)
	}

	started := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return FetchResult{}, fmt.Errorf("retrieve feed: %w", err)
	}
	defer resp.Body.Close()

	result := FetchResult{
		URL:          resp.Request.URL.String(),
		StatusCode:   resp.StatusCode,
		ETag:         strings.TrimSpace(resp.Header.Get("ETag")),
		LastModified: strings.TrimSpace(resp.Header.Get("Last-Modified")),
		ContentType:  strings.TrimSpace(resp.Header.Get("Content-Type")),
		FetchedAt:    time.Now().UTC(),
	}
	finish := func() FetchResult {
		result.Duration = time.Since(started)
		return result
	}

	if resp.StatusCode == http.StatusNotModified {
		result.NotModified = true
		return finish(), nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return finish(), &HTTPStatusError{StatusCode: resp.StatusCode}
	}
	if resp.ContentLength > c.maxBodyBytes {
		return finish(), ErrResponseTooLarge
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, c.maxBodyBytes+1))
	if err != nil {
		return finish(), fmt.Errorf("read feed retrieval response: %w", err)
	}
	if int64(len(body)) > c.maxBodyBytes {
		return finish(), ErrResponseTooLarge
	}
	result.Body = body
	return finish(), nil
}

func (c *Client) checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return ErrTooManyRedirects
	}
	if len(via) > 0 && strings.EqualFold(via[len(via)-1].URL.Scheme, "https") && strings.EqualFold(req.URL.Scheme, "http") {
		return ErrHTTPSDowngrade
	}
	if err := c.policy.ValidateURL(req.URL); err != nil {
		return err
	}
	if len(via) > 0 && !strings.EqualFold(via[len(via)-1].URL.Hostname(), req.URL.Hostname()) {
		req.Header.Del("If-None-Match")
		req.Header.Del("If-Modified-Since")
	}
	return nil
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
