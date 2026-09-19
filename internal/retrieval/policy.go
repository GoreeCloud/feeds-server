package retrieval

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
)

var (
	ErrDestinationBlocked  = errors.New("feed retrieval destination blocked")
	ErrEmbeddedCredentials = errors.New("feed retrieval URL must not contain credentials")
	ErrUnsupportedScheme   = errors.New("unsupported feed retrieval URL scheme")
)

// Resolver is the narrow DNS interface used by the retrieval transport.
// Keeping it explicit makes destination-policy behavior testable without
// performing external DNS queries.
type Resolver interface {
	LookupNetIP(context.Context, string, string) ([]netip.Addr, error)
}

// DestinationPolicy defines which remote feed destinations the retrieval
// transport may connect to. Private and special-use networks are denied by
// default and may be enabled only through explicit AllowedNetworks entries.
type DestinationPolicy struct {
	AllowPlainHTTP  bool
	AllowedNetworks []netip.Prefix
}

func (p DestinationPolicy) ValidateURL(u *url.URL) error {
	if u == nil || !u.IsAbs() || u.Host == "" || u.Hostname() == "" {
		return fmt.Errorf("%w: absolute HTTP(S) URL with host required", ErrDestinationBlocked)
	}
	if u.User != nil {
		return ErrEmbeddedCredentials
	}

	switch strings.ToLower(u.Scheme) {
	case "https":
	case "http":
		if !p.AllowPlainHTTP {
			return fmt.Errorf("%w: plaintext HTTP disabled", ErrUnsupportedScheme)
		}
	default:
		return fmt.Errorf("%w: %q", ErrUnsupportedScheme, u.Scheme)
	}

	if addr, err := netip.ParseAddr(u.Hostname()); err == nil {
		if err := p.validateAddress(addr); err != nil {
			return err
		}
	}
	return nil
}

func (p DestinationPolicy) resolveAllowed(ctx context.Context, resolver Resolver, host string) ([]netip.Addr, error) {
	if resolver == nil {
		resolver = net.DefaultResolver
	}

	if addr, err := netip.ParseAddr(host); err == nil {
		if err := p.validateAddress(addr); err != nil {
			return nil, err
		}
		return []netip.Addr{addr.Unmap()}, nil
	}

	addrs, err := resolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("resolve retrieval host %q: %w", host, err)
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("resolve retrieval host %q: no addresses", host)
	}

	out := make([]netip.Addr, 0, len(addrs))
	for _, addr := range addrs {
		addr = addr.Unmap()
		if err := p.validateAddress(addr); err != nil {
			return nil, fmt.Errorf("resolve retrieval host %q: %w", host, err)
		}
		out = append(out, addr)
	}
	return out, nil
}

func (p DestinationPolicy) validateAddress(addr netip.Addr) error {
	addr = addr.Unmap()
	if !addr.IsValid() || addr.Zone() != "" || addr.IsUnspecified() || addr.IsMulticast() {
		return fmt.Errorf("%w: address %q is not an eligible unicast destination", ErrDestinationBlocked, addr)
	}

	for _, prefix := range p.AllowedNetworks {
		if prefix.IsValid() && prefix.Contains(addr) {
			return nil
		}
	}

	if addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() {
		return fmt.Errorf("%w: address %q is private, loopback, or link-local", ErrDestinationBlocked, addr)
	}
	for _, prefix := range blockedSpecialUsePrefixes {
		if prefix.Contains(addr) {
			return fmt.Errorf("%w: address %q is special-use", ErrDestinationBlocked, addr)
		}
	}
	if !addr.IsGlobalUnicast() {
		return fmt.Errorf("%w: address %q is not global unicast", ErrDestinationBlocked, addr)
	}
	return nil
}

var blockedSpecialUsePrefixes = []netip.Prefix{
	mustPrefix("100.64.0.0/10"),
	mustPrefix("192.0.0.0/24"),
	mustPrefix("192.0.2.0/24"),
	mustPrefix("192.88.99.0/24"),
	mustPrefix("198.18.0.0/15"),
	mustPrefix("198.51.100.0/24"),
	mustPrefix("203.0.113.0/24"),
	mustPrefix("240.0.0.0/4"),
	mustPrefix("100::/64"),
	mustPrefix("2001:db8::/32"),
	mustPrefix("2001:10::/28"),
	mustPrefix("2001:20::/28"),
}

func mustPrefix(raw string) netip.Prefix {
	return netip.MustParsePrefix(raw)
}
