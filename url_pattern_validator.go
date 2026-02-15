package rmhttp

import (
	"fmt"
	"strings"
)

type urlPatternValidator struct{}

func newURLPatternValidator() urlPatternValidator {
	return urlPatternValidator{}
}

// validatePattern checks if a route pattern is valid according to RFC 3986
func (v urlPatternValidator) validate(pattern string) error {
	if pattern == "" {
		return fmt.Errorf("route pattern cannot be empty")
	}

	if len(pattern) > 256 {
		return fmt.Errorf("route pattern too long (max 256 chars): %s", pattern)
	}

	// Check for balanced braces in path parameters
	openBraces := strings.Count(pattern, "{")
	closeBraces := strings.Count(pattern, "}")
	if openBraces != closeBraces {
		return fmt.Errorf("unbalanced braces in pattern: %s", pattern)
	}

	// RFC 3986 defines the following character sets:
	// - unreserved: ALPHA / DIGIT / "-" / "." / "_" / "~"
	// - reserved: ":" / "/" / "?" / "#" / "[" / "]" / "@" / "!" / "$" / "&" / "'" / "(" / ")" / "*" / "+" / "," / ";" / "="
	// - percent-encoded: "%" HEXDIG HEXDIG
	//
	// For route patterns, we allow all RFC 3986 characters plus path parameter placeholders {param}
	// We also allow spaces (which will be percent-encoded as %20)

	var i int
	for i < len(pattern) {
		r := rune(pattern[i])
		if r == '%' {
			if i+2 >= len(pattern) {
				return fmt.Errorf("incomplete percent-encoding at end of pattern: %s", pattern)
			}
			// Validate that next two characters are hex digits
			hexChars := pattern[i+1 : i+3]
			if !v.isHexDigit(hexChars[0]) || !v.isHexDigit(hexChars[1]) {
				return fmt.Errorf(
					"invalid percent-encoding '%%%s' in pattern: %s",
					hexChars,
					pattern,
				)
			}
			i += 3
			continue
		}

		if v.isUnreserved(r) {
			i++
			continue
		}
		if v.isReserved(r) {
			i++
			continue
		}
		if r == '{' || r == '}' {
			i++
			continue
		}
		if r == ' ' {
			i++
			continue
		}

		return fmt.Errorf("invalid character '%c' (byte %d) in pattern: %s", r, i, pattern)
	}
	return nil
}

// isUnreserved checks if a character is in the RFC 3986 unreserved set
// unreserved = ALPHA / DIGIT / "-" / "." / "_" / "~"
func (v urlPatternValidator) isUnreserved(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '-' || r == '.' || r == '_' || r == '~'
}

// isReserved checks if a character is in the RFC 3986 reserved set
// reserved = ":" / "/" / "?" / "#" / "[" / "]" / "@" / "!" / "$" / "&" / "'" / "(" / ")" / "*" / "+" / "," / ";" / "="
func (v urlPatternValidator) isReserved(r rune) bool {
	return r == ':' || r == '/' || r == '?' || r == '#' || r == '[' || r == ']' || r == '@' ||
		r == '!' || r == '$' || r == '&' || r == '\'' || r == '(' || r == ')' ||
		r == '*' || r == '+' || r == ',' || r == ';' || r == '='
}

// isHexDigit checks if a character is a valid hexadecimal digit
func (v urlPatternValidator) isHexDigit(r byte) bool {
	return (r >= '0' && r <= '9') ||
		(r >= 'a' && r <= 'f') ||
		(r >= 'A' && r <= 'F')
}
