package tools

import (
	"encoding/base64"
	"regexp"
	"strings"
)

// The rules here align with imagor-studio's converter
// (server/internal/imagortemplate/converter.go), so a path this tool builds and a
// path that project builds are encoded the same way. That matters because they
// are meant to be interchangeable: a template authored in one should render in
// the other.
//
// The two cases are different on purpose.
//
// TEXT takes a whitelist. Only characters that are unambiguously inert are left
// alone, and everything else is encoded, so a value that is entirely ordinary
// stays readable while anything at all unusual is safe. A whitelist is the right
// shape here because the input is arbitrary: it comes from a title, a product
// name, a field in someone's database.
//
// A SOURCE takes a character set plus reserved prefixes. A source is nearly always
// a path or a URL, which legitimately contains '/', ':', '.' and '=', so a
// whitelist would encode almost every real source and make every path opaque. The
// list covers the characters that would change how the path is read, and the
// prefixes cover a source whose own text would be read as a command.

// simpleText is the set of characters that need no encoding in a text argument.
var simpleText = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// needingSourceEncoding are the characters that change how a path is read if a
// source contains them: a space ends the path, ? starts a query string, # starts
// a fragment, & separates query parameters, and ( ) , are the filter syntax.
const needingSourceEncoding = " ?#&(),"

// reservedSourcePrefixes are the words imagor reads as a command rather than as
// the start of a source. A source that happens to begin with one is encoded even
// though it contains no special character at all.
var reservedSourcePrefixes = []string{
	"trim/", "meta/", "fit-in/", "stretch/",
	"top/", "left/", "right/", "bottom/", "center/", "smart/",
}

// encodeTextValue encodes a text argument only when it is not trivially safe.
func encodeTextValue(text string) string {
	if text == "" || simpleText.MatchString(text) {
		return text
	}
	// Pango markup is interpreted by the renderer, so a value containing markup
	// characters has to be escaped as well as encoded. Measured: an unescaped &
	// is answered with a 406 invalid markup.
	escaped := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(text)
	return "b64:" + base64.RawURLEncoding.EncodeToString([]byte(escaped))
}

// decodeTextValue undoes encodeTextValue.
func decodeTextValue(v string) string {
	if !strings.HasPrefix(v, "b64:") {
		return v
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(v, "b64:"))
	if err != nil {
		return v
	}
	return strings.NewReplacer("&amp;", "&", "&lt;", "<", "&gt;", ">").Replace(string(raw))
}

// encodeSourceValue encodes a source only when leaving it alone would change how
// the path is read.
func encodeSourceValue(source string) string {
	if source == "" {
		return source
	}
	if strings.ContainsAny(source, needingSourceEncoding) {
		return "b64:" + base64.RawURLEncoding.EncodeToString([]byte(source))
	}
	for _, p := range reservedSourcePrefixes {
		if strings.HasPrefix(source, p) {
			return "b64:" + base64.RawURLEncoding.EncodeToString([]byte(source))
		}
	}
	return source
}
