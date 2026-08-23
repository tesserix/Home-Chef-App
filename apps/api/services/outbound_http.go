package services

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"time"
)

var nonPublicPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001:db8::/32"),
}

// ValidatePublicHTTPSURL rejects URLs that could reach local infrastructure or
// disclose credentials over plaintext. Hostnames are resolved and every answer
// must be public; the secure transport repeats this check at dial time.
func ValidatePublicHTTPSURL(ctx context.Context, rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("URL must use HTTPS and include a host")
	}
	if u.User != nil {
		return errors.New("URL credentials are not allowed")
	}
	_, err = resolvePublicIPs(ctx, u.Hostname())
	return err
}

// NewPublicHTTPSClient returns a direct HTTP client whose dialer validates and
// pins each resolved public IP, preventing DNS rebinding and redirect SSRF.
func NewPublicHTTPSClient(timeout time.Duration) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = dialPublicContext
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func dialPublicContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid outbound address: %w", err)
	}
	ips, err := resolvePublicIPs(ctx, host)
	if err != nil {
		return nil, err
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	var dialErr error
	for _, ip := range ips {
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
		dialErr = err
	}
	return nil, fmt.Errorf("connect to public provider endpoint: %w", dialErr)
}

func resolvePublicIPs(ctx context.Context, host string) ([]netip.Addr, error) {
	if addr, err := netip.ParseAddr(host); err == nil {
		addr = addr.Unmap()
		if !isSafeOutboundIP(addr) {
			return nil, errors.New("private or reserved addresses are not allowed")
		}
		return []netip.Addr{addr}, nil
	}

	resolved, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("resolve provider host: %w", err)
	}
	if len(resolved) == 0 {
		return nil, errors.New("provider host has no IP addresses")
	}
	result := make([]netip.Addr, 0, len(resolved))
	for _, addr := range resolved {
		addr = addr.Unmap()
		if !isSafeOutboundIP(addr) {
			return nil, errors.New("provider host resolves to a private or reserved address")
		}
		result = append(result, addr)
	}
	return result, nil
}

func isSafeOutboundIP(addr netip.Addr) bool {
	if !addr.IsValid() || !addr.IsGlobalUnicast() || addr.IsPrivate() || addr.IsLoopback() ||
		addr.IsLinkLocalUnicast() || addr.IsMulticast() || addr.IsUnspecified() {
		return false
	}
	for _, prefix := range nonPublicPrefixes {
		if prefix.Contains(addr) {
			return false
		}
	}
	return true
}
