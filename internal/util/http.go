package util

import (
	"fmt"
	"net"
	"net/http"
	"slices"
	"strings"

	"github.com/GoBetterAuth/go-better-auth/models"
)

// MaskIP masks the last octet of an IPv4 address for GDPR compliance
// Example: "1.2.3.4" -> "1.2.3.x"
// For IPv6 or other formats, returns the original string unchanged
func MaskIP(ip string) string {
	if ip == "" {
		return ""
	}

	// Handle IPv4 addresses
	if strings.Count(ip, ":") == 0 && strings.Count(ip, ".") == 3 {
		parts := strings.Split(ip, ".")
		if len(parts) == 4 {
			parts[3] = "x"
			return strings.Join(parts, ".")
		}
	}

	// For IPv6 or other formats, return original
	// (masking IPv6 is more complex and context-dependent)
	return ip
}

// ExtractClientIP extracts the client IP address from various sources accurately
func ExtractClientIP(logger models.Logger, req *http.Request, trustedHeaders []string, trustedProxies []string) (net.IP, error) {
	host := getHost(req.RemoteAddr)
	remoteIP := net.ParseIP(host)
	if remoteIP == nil {
		return nil, fmt.Errorf("invalid remote IP: %s", host)
	}

	// SECURITY GATE: If no trusted proxies are configured,
	// we NEVER trust headers to prevent trivial spoofing.
	if len(trustedProxies) == 0 {
		return remoteIP, nil
	}

	// Verify if the immediate caller is a trusted proxy
	if !isIPTrusted(remoteIP, trustedProxies) {
		// Connection is from an untrusted source; use their direct IP
		return remoteIP, nil
	}

	// Build the header priority list (User defined + Defaults)
	headers := make([]string, 0, len(trustedHeaders)+2)
	headers = append(headers, trustedHeaders...)
	for _, d := range []string{"X-Forwarded-For"} {
		if !slices.Contains(headers, d) {
			headers = append(headers, d)
		}
	}

	// Traverse headers to find the first valid IP
	for _, h := range headers {
		val := req.Header.Get(h)
		if val == "" {
			continue
		}

		// Handle list headers (like X-Forwarded-For: client, proxy1)
		parts := strings.SplitSeq(val, ",")
		for part := range parts {
			ipStr := strings.TrimSpace(part)

			// Handle cases where the header might contain a port (e.g., 1.2.3.4:8080)
			if sh, _, err := net.SplitHostPort(ipStr); err == nil {
				ipStr = sh
			}

			if ip := net.ParseIP(ipStr); ip != nil {
				return ip, nil
			}
		}
	}

	return remoteIP, nil
}

func getHost(remoteAddr string) string {
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	return remoteAddr // It's an IP without port
}

// isIPTrusted checks if an IP matches a list of exact IPs or CIDR ranges.
func isIPTrusted(ip net.IP, trustedProxies []string) bool {
	for _, trusted := range trustedProxies {
		// Check for exact IP match
		if ip.String() == trusted {
			return true
		}

		// Check for CIDR range match (e.g., 10.0.0.0/8)
		_, subnet, err := net.ParseCIDR(trusted)
		if err == nil && subnet.Contains(ip) {
			return true
		}
	}

	return false
}
