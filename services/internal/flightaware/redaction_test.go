package flightaware

import (
	"strings"
	"testing"
)

func TestRedactHeadersRemovesSensitiveHeaderValues(t *testing.T) {
	headers := map[string][]string{
		"x-apikey":      {"flightaware-secret"},
		"Authorization": {"Bearer token-value"},
		"Cookie":        {"sessionid=abc"},
		"Accept":        {"application/json"},
	}

	redacted := RedactHeaders(headers, []string{"flightaware-secret", "token-value", "abc"})

	if redacted["x-apikey"][0] != RedactedValue || redacted["Authorization"][0] != RedactedValue || redacted["Cookie"][0] != RedactedValue {
		t.Fatalf("sensitive headers were not redacted: %#v", redacted)
	}
	if redacted["Accept"][0] != "application/json" {
		t.Fatalf("Accept = %q, want application/json", redacted["Accept"][0])
	}
}

func TestRedactLogTextRemovesRawSecretsAndCredentialFields(t *testing.T) {
	input := `request x-apikey=flightaware-secret Authorization="Bearer token-value" Cookie=sessionid=abc api_key=flightaware-secret`
	redacted := RedactLogText(input, []string{"flightaware-secret", "token-value", "abc"})

	for _, leaked := range []string{"flightaware-secret", "token-value", "sessionid=abc", "api_key=flightaware-secret"} {
		if strings.Contains(redacted, leaked) {
			t.Fatalf("redacted text leaked %q: %s", leaked, redacted)
		}
	}
	if !strings.Contains(redacted, RedactedValue) {
		t.Fatalf("redacted text = %q, want redaction marker", redacted)
	}
}
