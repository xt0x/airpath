package flightaware

import (
	"regexp"
	"strings"
)

const RedactedValue = "[REDACTED]"

var sensitiveHeaderNames = map[string]struct{}{
	"authorization": {},
	"cookie":        {},
	"x-apikey":      {},
}

var credentialFieldPattern = regexp.MustCompile(`(?i)(x-apikey|authorization|cookie|api[_-]?key)\s*=\s*("[^"]*"|[^\s]+)`)

func RedactHeaders(headers map[string][]string, rawSecrets []string) map[string][]string {
	redacted := make(map[string][]string, len(headers))
	for name, values := range headers {
		copied := append([]string(nil), values...)
		if _, ok := sensitiveHeaderNames[strings.ToLower(name)]; ok {
			for index := range copied {
				copied[index] = RedactedValue
			}
		} else {
			for index, value := range copied {
				copied[index] = RedactLogText(value, rawSecrets)
			}
		}
		redacted[name] = copied
	}
	return redacted
}

func RedactLogText(text string, rawSecrets []string) string {
	redacted := credentialFieldPattern.ReplaceAllString(text, "$1="+RedactedValue)
	for _, secret := range rawSecrets {
		secret = strings.TrimSpace(secret)
		if secret == "" {
			continue
		}
		redacted = strings.ReplaceAll(redacted, secret, RedactedValue)
	}
	return redacted
}
