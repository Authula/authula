package util

import (
	"net"
	"strings"
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

// ExtractClientIP extracts the client IP address from various sources
// Checks X-Forwarded-For, X-Real-IP headers and falls back to RemoteAddr
func ExtractClientIP(xForwardedFor string, realIp string, remoteAddr string) string {
	// Check X-Forwarded-For header first (may contain multiple IPs)
	if xForwardedFor != "" {
		// Take the first IP from the list
		ips := strings.Split(xForwardedFor, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if realIp != "" {
		return realIp
	}

	// Fall back to RemoteAddr
	if remoteAddr != "" {
		// RemoteAddr might include port, extract just the IP
		ip, _, err := net.SplitHostPort(remoteAddr)
		if err != nil {
			return remoteAddr
		}
		return ip
	}

	return ""
}
