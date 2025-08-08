package handler

import (
	"net"
	"net/url"
	"strings"
)

func isValidURL(testURL string) bool {
	u, err := url.ParseRequestURI(testURL)
	if err != nil {
		return false
	}

	allowedSchemes := map[string]bool{
		"http":  true,
		"https": true,
	}
	if !allowedSchemes[strings.ToLower(u.Scheme)] {
		return false
	}

	if u.Host == "" {
		return false
	}

	hostname := u.Hostname()
	if hostname == "" {
		return false
	}

	ip := net.ParseIP(hostname)
	if ip != nil {
		if isPrivateOrReservedIP(ip) {
			return false
		}
	} else {
		lowerHostname := strings.ToLower(hostname)
		forbiddenHosts := map[string]bool{
			"localhost":     true,
			"localhost.":    true, // FQDN variant
			"local":         true,
			"broadcasthost": true,
		}
		if forbiddenHosts[lowerHostname] {
			return false
		}

		if strings.HasSuffix(lowerHostname, ".local") ||
			strings.HasSuffix(lowerHostname, ".localhost") ||
			strings.HasSuffix(lowerHostname, ".internal") {
			return false
		}
	}

	return true
}

// isPrivateOrReservedIP checks if an IP is in private or reserved ranges
func isPrivateOrReservedIP(ip net.IP) bool {
	privateRanges := []string{
		"127.0.0.0/8",    // Loopback
		"10.0.0.0/8",     // Private Class A
		"172.16.0.0/12",  // Private Class B
		"192.168.0.0/16", // Private Class C
		"169.254.0.0/16", // Link-local
		"::1/128",        // IPv6 loopback
		"fc00::/7",       // IPv6 unique local
		"fe80::/10",      // IPv6 link-local
		"0.0.0.0/8",      // Current network
		"224.0.0.0/4",    // Multicast
		"240.0.0.0/4",    // Reserved
	}

	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}

	return false
}
