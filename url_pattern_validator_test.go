package rmhttp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ------------------------------------------------------------------------------------------------
// URL PATTERN VALIDATOR TESTS
// ------------------------------------------------------------------------------------------------

// Test_urlPatternValidator_EmptyPattern tests that an empty pattern returns an error.
func Test_urlPatternValidator_EmptyPattern(t *testing.T) {
	v := newURLPatternValidator()

	err := v.validate("")

	assert.EqualError(t, err, "route pattern cannot be empty")
}

// Test_urlPatternValidator_MaxLength tests that patterns over 256 characters return an error.
func Test_urlPatternValidator_MaxLength(t *testing.T) {
	v := newURLPatternValidator()

	longPattern := "a"
	for i := 0; i < 256; i++ {
		longPattern += "a"
	}

	err := v.validate(longPattern)
	assert.EqualError(t, err, "route pattern too long (max 256 chars): "+longPattern)
}

// Test_urlPatternValidator_BalancedBraces tests that unbalanced braces return an error.
func Test_urlPatternValidator_BalancedBraces(t *testing.T) {
	v := newURLPatternValidator()

	t.Run("unbalanced opening brace", func(t *testing.T) {
		err := v.validate("/path/{id")
		assert.EqualError(t, err, "unbalanced braces in pattern: /path/{id")
	})

	t.Run("unbalanced closing brace", func(t *testing.T) {
		err := v.validate("/path/id}")
		assert.EqualError(t, err, "unbalanced braces in pattern: /path/id}")
	})

	t.Run("multiple opening braces", func(t *testing.T) {
		err := v.validate("/path/{id}/{name")
		assert.EqualError(t, err, "unbalanced braces in pattern: /path/{id}/{name")
	})

	t.Run("multiple closing braces", func(t *testing.T) {
		err := v.validate("/path/id}}")
		assert.EqualError(t, err, "unbalanced braces in pattern: /path/id}}")
	})

	t.Run("balanced braces", func(t *testing.T) {
		err := v.validate("/path/{id}")
		assert.NoError(t, err)
	})

	t.Run("multiple balanced braces", func(t *testing.T) {
		err := v.validate("/path/{id}/{name}")
		assert.NoError(t, err)
	})
}

// Test_urlPatternValidator_UnreservedCharacters tests RFC 3986 unreserved characters.
// unreserved = ALPHA / DIGIT / "-" / "." / "_" / "~"
func Test_urlPatternValidator_UnreservedCharacters(t *testing.T) {
	v := newURLPatternValidator()

	t.Run("lowercase letters", func(t *testing.T) {
		err := v.validate("/abcdefghijklmnopqrstuvwxyz")
		assert.NoError(t, err)
	})

	t.Run("uppercase letters", func(t *testing.T) {
		err := v.validate("/ABCDEFGHIJKLMNOPQRSTUVWXYZ")
		assert.NoError(t, err)
	})

	t.Run("digits", func(t *testing.T) {
		err := v.validate("/0123456789")
		assert.NoError(t, err)
	})

	t.Run("hyphen", func(t *testing.T) {
		err := v.validate("/path-name")
		assert.NoError(t, err)
	})

	t.Run("dot", func(t *testing.T) {
		err := v.validate("/file.name")
		assert.NoError(t, err)
	})

	t.Run("underscore", func(t *testing.T) {
		err := v.validate("/path_name")
		assert.NoError(t, err)
	})

	t.Run("tilde", func(t *testing.T) {
		err := v.validate("/path~name")
		assert.NoError(t, err)
	})

	t.Run("all unreserved characters together", func(t *testing.T) {
		err := v.validate("/abc-ABC_123._~")
		assert.NoError(t, err)
	})
}

// Test_urlPatternValidator_ReservedCharacters tests RFC 3986 reserved characters.
// reserved = ":" / "/" / "?" / "#" / "[" / "]" / "@" / "!" / "$" / "&" / "'" / "(" / ")" / "*" / "+" / "," / ";" / "="
func Test_urlPatternValidator_ReservedCharacters(t *testing.T) {
	v := newURLPatternValidator()

	t.Run("colon", func(t *testing.T) {
		err := v.validate("/path:name")
		assert.NoError(t, err)
	})

	t.Run("slash", func(t *testing.T) {
		err := v.validate("/path/name")
		assert.NoError(t, err)
	})

	t.Run("question mark", func(t *testing.T) {
		err := v.validate("/path?name")
		assert.NoError(t, err)
	})

	t.Run("hash", func(t *testing.T) {
		err := v.validate("/path#name")
		assert.NoError(t, err)
	})

	t.Run("square brackets", func(t *testing.T) {
		err := v.validate("/path[name]")
		assert.NoError(t, err)
	})

	t.Run("at sign", func(t *testing.T) {
		err := v.validate("/path@name")
		assert.NoError(t, err)
	})

	t.Run("exclamation mark", func(t *testing.T) {
		err := v.validate("/path!name")
		assert.NoError(t, err)
	})

	t.Run("dollar sign", func(t *testing.T) {
		err := v.validate("/path$name")
		assert.NoError(t, err)
	})

	t.Run("ampersand", func(t *testing.T) {
		err := v.validate("/path&name")
		assert.NoError(t, err)
	})

	t.Run("single quote", func(t *testing.T) {
		err := v.validate("/path'name")
		assert.NoError(t, err)
	})

	t.Run("parentheses", func(t *testing.T) {
		err := v.validate("/path(name)")
		assert.NoError(t, err)
	})

	t.Run("asterisk", func(t *testing.T) {
		err := v.validate("/path*name")
		assert.NoError(t, err)
	})

	t.Run("plus", func(t *testing.T) {
		err := v.validate("/path+name")
		assert.NoError(t, err)
	})

	t.Run("comma", func(t *testing.T) {
		err := v.validate("/path,name")
		assert.NoError(t, err)
	})

	t.Run("semicolon", func(t *testing.T) {
		err := v.validate("/path;name")
		assert.NoError(t, err)
	})

	t.Run("equals", func(t *testing.T) {
		err := v.validate("/path=name")
		assert.NoError(t, err)
	})

	t.Run("all reserved characters together", func(t *testing.T) {
		err := v.validate("/path:/?#[]@!$&'()*+,;=name")
		assert.NoError(t, err)
	})
}

// Test_urlPatternValidator_PercentEncoding tests RFC 3986 percent-encoding.
// percent-encoded = "%" HEXDIG HEXDIG
func Test_urlPatternValidator_PercentEncoding(t *testing.T) {
	v := newURLPatternValidator()

	t.Run("valid percent-encoding lowercase", func(t *testing.T) {
		err := v.validate("/path%20name")
		assert.NoError(t, err)
	})

	t.Run("valid percent-encoding uppercase", func(t *testing.T) {
		err := v.validate("/path%2Fname")
		assert.NoError(t, err)
	})

	t.Run("valid percent-encoding mixed case", func(t *testing.T) {
		err := v.validate("/path%2fName%3A")
		assert.NoError(t, err)
	})

	t.Run("multiple percent-encodings", func(t *testing.T) {
		err := v.validate("/path%20name%2Ftest%3Dvalue")
		assert.NoError(t, err)
	})

	t.Run("percent-encoding at end", func(t *testing.T) {
		err := v.validate("/path%20")
		assert.NoError(t, err)
	})

	t.Run("incomplete percent-encoding (missing one digit)", func(t *testing.T) {
		err := v.validate("/path%2")
		assert.EqualError(t, err, "incomplete percent-encoding at end of pattern: /path%2")
	})

	t.Run("incomplete percent-encoding (missing two digits)", func(t *testing.T) {
		err := v.validate("/path%")
		assert.EqualError(t, err, "incomplete percent-encoding at end of pattern: /path%")
	})

	t.Run("invalid percent-encoding (non-hex character)", func(t *testing.T) {
		err := v.validate("/path%2g")
		assert.EqualError(t, err, "invalid percent-encoding '%2g' in pattern: /path%2g")
	})

	t.Run("invalid percent-encoding (non-hex character second digit)", func(t *testing.T) {
		err := v.validate("/path%g2")
		assert.EqualError(t, err, "invalid percent-encoding '%g2' in pattern: /path%g2")
	})
}

// Test_urlPatternValidator_PathParameters tests path parameter placeholders.
func Test_urlPatternValidator_PathParameters(t *testing.T) {
	v := newURLPatternValidator()

	t.Run("single path parameter", func(t *testing.T) {
		err := v.validate("/users/{id}")
		assert.NoError(t, err)
	})

	t.Run("multiple path parameters", func(t *testing.T) {
		err := v.validate("/users/{id}/posts/{postId}")
		assert.NoError(t, err)
	})

	t.Run("path parameter with hyphens", func(t *testing.T) {
		err := v.validate("/users/{user-id}")
		assert.NoError(t, err)
	})

	t.Run("path parameter with underscores", func(t *testing.T) {
		err := v.validate("/users/{user_id}")
		assert.NoError(t, err)
	})
}

// Test_urlPatternValidator_Spaces tests that spaces are allowed (will be percent-encoded).
func Test_urlPatternValidator_Spaces(t *testing.T) {
	v := newURLPatternValidator()

	err := v.validate("/path/with spaces")
	assert.NoError(t, err)
}

// Test_urlPatternValidator_InvalidCharacters tests characters that are not allowed.
func Test_urlPatternValidator_InvalidCharacters(t *testing.T) {
	v := newURLPatternValidator()

	t.Run("backtick", func(t *testing.T) {
		err := v.validate("/path`name")
		assert.EqualError(t, err, "invalid character '`' (byte 5) in pattern: /path`name")
	})

	t.Run("backslash", func(t *testing.T) {
		err := v.validate("/path\\name")
		assert.EqualError(t, err, "invalid character '\\' (byte 5) in pattern: /path\\name")
	})

	t.Run("pipe", func(t *testing.T) {
		err := v.validate("/path|name")
		assert.EqualError(t, err, "invalid character '|' (byte 5) in pattern: /path|name")
	})

	t.Run("angle brackets", func(t *testing.T) {
		err := v.validate("/path<name>")
		assert.EqualError(t, err, "invalid character '<' (byte 5) in pattern: /path<name>")
	})

	t.Run("curly braces (not path params - unbalanced)", func(t *testing.T) {
		err := v.validate("/path{name")
		assert.EqualError(t, err, "unbalanced braces in pattern: /path{name")
	})

	t.Run("vertical bar", func(t *testing.T) {
		err := v.validate("/path|name")
		assert.EqualError(t, err, "invalid character '|' (byte 5) in pattern: /path|name")
	})

	t.Run("caret", func(t *testing.T) {
		err := v.validate("/path^name")
		assert.EqualError(t, err, "invalid character '^' (byte 5) in pattern: /path^name")
	})

	t.Run("grave accent", func(t *testing.T) {
		err := v.validate("/path`name")
		assert.EqualError(t, err, "invalid character '`' (byte 5) in pattern: /path`name")
	})

	t.Run("space in middle", func(t *testing.T) {
		// This should actually pass - spaces are allowed
		err := v.validate("/path name")
		assert.NoError(t, err)
	})
}

// Test_urlPatternValidator_ComplexPatterns tests complex real-world URL patterns.
func Test_urlPatternValidator_ComplexPatterns(t *testing.T) {
	v := newURLPatternValidator()

	t.Run("REST API endpoint", func(t *testing.T) {
		err := v.validate("/api/v1/users/{id}/posts/{postId}")
		assert.NoError(t, err)
	})

	t.Run("pattern with query-like syntax", func(t *testing.T) {
		err := v.validate("/search?q={query}")
		assert.NoError(t, err)
	})

	t.Run("pattern with hash fragment", func(t *testing.T) {
		err := v.validate("/page/{id}#section")
		assert.NoError(t, err)
	})

	t.Run("pattern with special characters", func(t *testing.T) {
		err := v.validate("/files/{filename}.txt")
		assert.NoError(t, err)
	})

	t.Run("pattern with percent-encoded characters", func(t *testing.T) {
		err := v.validate("/files/{path}/download%20file")
		assert.NoError(t, err)
	})

	t.Run("complex nested pattern", func(t *testing.T) {
		err := v.validate("/api/{version}/users/{userId}/posts/{postId}/comments/{commentId}")
		assert.NoError(t, err)
	})
}

// Test_urlPatternValidator_RFC3986Examples tests examples from RFC 3986.
func Test_urlPatternValidator_RFC3986Examples(t *testing.T) {
	v := newURLPatternValidator()

	// RFC 3986 Section 2.3: Unreserved Characters
	t.Run("unreserved characters from RFC", func(t *testing.T) {
		err := v.validate("/abcABC012-._~")
		assert.NoError(t, err)
	})

	// RFC 3986 Section 2.2: Reserved Characters
	t.Run("reserved characters from RFC", func(t *testing.T) {
		err := v.validate(":/?#[]@!$&'()*+,;=")
		assert.NoError(t, err)
	})

	// RFC 3986 Section 2.1: Percent-Encoding
	t.Run("percent-encoded examples from RFC", func(t *testing.T) {
		err := v.validate("/%7Euser/%20space/%3Fquery")
		assert.NoError(t, err)
	})
}

// Test_urlPatternValidator_ValidatorMethods tests the individual validator methods.
func Test_urlPatternValidator_ValidatorMethods(t *testing.T) {
	v := newURLPatternValidator()

	t.Run("isUnreserved", func(t *testing.T) {
		assert.True(t, v.isUnreserved('a'), "lowercase 'a' should be unreserved")
		assert.True(t, v.isUnreserved('z'), "lowercase 'z' should be unreserved")
		assert.True(t, v.isUnreserved('A'), "uppercase 'A' should be unreserved")
		assert.True(t, v.isUnreserved('Z'), "uppercase 'Z' should be unreserved")
		assert.True(t, v.isUnreserved('0'), "digit '0' should be unreserved")
		assert.True(t, v.isUnreserved('9'), "digit '9' should be unreserved")
		assert.True(t, v.isUnreserved('-'), "hyphen should be unreserved")
		assert.True(t, v.isUnreserved('.'), "dot should be unreserved")
		assert.True(t, v.isUnreserved('_'), "underscore should be unreserved")
		assert.True(t, v.isUnreserved('~'), "tilde should be unreserved")

		assert.False(t, v.isUnreserved(' '), "space should not be unreserved")
		assert.False(t, v.isUnreserved('/'), "slash should not be unreserved")
		assert.False(t, v.isUnreserved('%'), "percent should not be unreserved")
	})

	t.Run("isReserved", func(t *testing.T) {
		assert.True(t, v.isReserved(':'), "colon should be reserved")
		assert.True(t, v.isReserved('/'), "slash should be reserved")
		assert.True(t, v.isReserved('?'), "question mark should be reserved")
		assert.True(t, v.isReserved('#'), "hash should be reserved")
		assert.True(t, v.isReserved('['), "left bracket should be reserved")
		assert.True(t, v.isReserved(']'), "right bracket should be reserved")
		assert.True(t, v.isReserved('@'), "at sign should be reserved")
		assert.True(t, v.isReserved('!'), "exclamation should be reserved")
		assert.True(t, v.isReserved('$'), "dollar should be reserved")
		assert.True(t, v.isReserved('&'), "ampersand should be reserved")
		assert.True(t, v.isReserved('\''), "single quote should be reserved")
		assert.True(t, v.isReserved('('), "left paren should be reserved")
		assert.True(t, v.isReserved(')'), "right paren should be reserved")
		assert.True(t, v.isReserved('*'), "asterisk should be reserved")
		assert.True(t, v.isReserved('+'), "plus should be reserved")
		assert.True(t, v.isReserved(','), "comma should be reserved")
		assert.True(t, v.isReserved(';'), "semicolon should be reserved")
		assert.True(t, v.isReserved('='), "equals should be reserved")

		assert.False(t, v.isReserved('a'), "letter should not be reserved")
		assert.False(t, v.isReserved('-'), "hyphen should not be reserved")
		assert.False(t, v.isReserved('%'), "percent should not be reserved")
	})

	t.Run("isHexDigit", func(t *testing.T) {
		assert.True(t, v.isHexDigit('0'), "'0' should be hex")
		assert.True(t, v.isHexDigit('9'), "'9' should be hex")
		assert.True(t, v.isHexDigit('a'), "'a' should be hex")
		assert.True(t, v.isHexDigit('f'), "'f' should be hex")
		assert.True(t, v.isHexDigit('A'), "'A' should be hex")
		assert.True(t, v.isHexDigit('F'), "'F' should be hex")

		assert.False(t, v.isHexDigit('g'), "'g' should not be hex")
		assert.False(t, v.isHexDigit('z'), "'z' should not be hex")
		assert.False(t, v.isHexDigit('-'), "hyphen should not be hex")
	})
}
