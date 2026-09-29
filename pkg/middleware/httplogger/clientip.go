package httplogger

import (
	"net/http"
	"net/netip"
	"strings"
)

// clientIP resolves the client IP address for r. It first selects the leftmost (most client-facing)
// usable entry of X-Forwarded-For, then falls back to X-Real-IP, and finally to the connection's
// peer address. Every candidate is validated and normalized with net/netip: empty and RFC 7239
// "unknown" entries are skipped, ports and IPv6 brackets are stripped, and zone IDs and
// IPv4-mapped addresses are normalized before logging.
//
// It returns "" when no candidate yields a usable address; the caller must still emit its log
// entry.
//
// Proxy headers are trusted from any peer here. Deployments that must only believe headers
// forwarded by known proxies should resolve the client IP in a trusted-proxy-aware
// middleware and log that value instead.
func clientIP(r *http.Request) string {
	// All X-Forwarded-For header lines are treated as one comma-separated chain so a client cannot
	// shadow a line with one of its own.
	for _, line := range r.Header.Values("X-Forwarded-For") {
		for entry := range strings.SplitSeq(line, ",") {
			entry = strings.TrimSpace(entry)
			if entry == "" || strings.EqualFold(entry, "unknown") {
				continue
			}
			if _, text, ok := parseAddr(entry); ok {
				return text
			}
		}
	}
	if ip, ok := realIPHeader(r); ok {
		return ip
	}
	_, text, ok := parseAddr(r.RemoteAddr)
	if !ok {
		return ""
	}
	return text
}

// realIPHeader returns the X-Real-IP header value when it is present and parses as an IP address.
func realIPHeader(r *http.Request) (string, bool) {
	value := r.Header.Get("X-Real-Ip")
	if value == "" {
		return "", false
	}
	_, text, ok := parseAddr(strings.TrimSpace(value))
	return text, ok
}

// parseAddr parses s as an IP address, with or without a port. It returns the normalized address
// plus its loggable text. The text is an allocation-free substring of s whenever no
// normalization is needed; it is rebuilt from the parsed address only when an
// IPv4-mapped address was unmapped (rare slow path).
//
// The port/brackets/zone are removed with cheap structural slicing first, so a single
// netip.ParseAddr validates the bare address. This avoids the cost of
// netip.ParseAddrPort on the common host:port form. Consequence:
// the port text is not range-checked ("1.2.3.4:99999" yields
// "1.2.3.4"); the address itself is always fully validated,
// and anything else fails to parse and yields ok=false.
func parseAddr(s string) (netip.Addr, string, bool) {
	host := trimHost(s)
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}, "", false
	}
	if unmapped := addr.Unmap(); unmapped != addr {
		return unmapped, unmapped.String(), true
	}
	return addr, host, true
}

// trimHost returns s with any port, IPv6 brackets, and zone ID removed, as a substring of s. It
// relies only on structural markers (brackets, colon count) and never validates;
// netip.ParseAddr does that afterward. A bare IPv6 that carries a port without
// brackets has 2+ colons and is left whole, so ParseAddr then rejects it
// (fail-closed).
func trimHost(s string) string {
	var host string
	switch {
	case s == "":
		return ""
	case s[0] == '[':
		if end := strings.IndexByte(s, ']'); end > 0 {
			host = s[1:end]
		} else {
			host = s
		}
	default:
		first := strings.IndexByte(s, ':')
		if first >= 0 && strings.IndexByte(s[first+1:], ':') < 0 {
			// Exactly one colon: an unbracketed host:port pair.
			host = s[:first]
		} else {
			// No colon (bare IPv4) or 2+ colons (bare IPv6, possibly invalid).
			host = s
		}
	}
	if zone := strings.IndexByte(host, '%'); zone >= 0 {
		host = host[:zone]
	}
	return host
}
