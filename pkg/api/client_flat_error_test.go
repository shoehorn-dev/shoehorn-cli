package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode"

	"pgregory.net/rapid"
)

func serveBody(t *testing.T, status int, body string) error {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()
	return NewClient(server.URL).Get(context.Background(), "/api/v1/me", nil)
}

// The platform's auth middleware answers with a flat envelope. After an
// identity provider switch every command got this, and the CLI printed the raw
// JSON cut off at 200 characters instead of the reason.
func TestClient_FlatErrorEnvelopeIsParsed(t *testing.T) {
	body := `{"code":"IDENTITY_CONFLICT","email":"dana@example.com","error":"identity_conflict","message":"This email already belongs to a different sign-in identity in this organization. An administrator must link the accounts."}`

	err := serveBody(t, http.StatusConflict, body)

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want *APIError", err)
	}
	if apiErr.StatusCode != 409 || apiErr.Code != "IDENTITY_CONFLICT" {
		t.Fatalf("got %d/%q, want 409/IDENTITY_CONFLICT", apiErr.StatusCode, apiErr.Code)
	}
	if !strings.HasPrefix(apiErr.Message, "This email already belongs") || strings.Contains(apiErr.Message, "{") {
		t.Fatalf("message = %q, want the human message, not raw JSON", apiErr.Message)
	}
	if strings.Contains(apiErr.Message, "truncated") {
		t.Fatalf("message was truncated: %q", apiErr.Message)
	}
}

func TestClient_NestedErrorEnvelopeStillParsed(t *testing.T) {
	err := serveBody(t, http.StatusNotFound, `{"error":{"message":"entity not found","code":"NOT_FOUND"}}`)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "NOT_FOUND" || apiErr.Message != "entity not found" {
		t.Fatalf("err = %v, want 404 NOT_FOUND entity not found", err)
	}
}

func TestClient_NonJSONErrorBodyStaysRaw(t *testing.T) {
	err := serveBody(t, http.StatusBadGateway, `<html>bad gateway</html>`)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Message != "<html>bad gateway</html>" || apiErr.Code != "" {
		t.Fatalf("err = %v, want the raw body with no code", err)
	}
}

// A flat body whose error is just a word, like the platform's 401, still
// gives a readable message.
func TestClient_FlatErrorWithoutMessageUsesTheErrorWord(t *testing.T) {
	err := serveBody(t, http.StatusForbidden, `{"error":"forbidden"}`)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Message != "forbidden" {
		t.Fatalf("err = %v, want message forbidden", err)
	}
}

// Any JSON object decodes into the nested struct, so an object with no
// "error" key must still fall through to the other shapes.
func TestClient_ObjectWithoutErrorKeyIsNotEmpty(t *testing.T) {
	for body, want := range map[string]string{
		`{"message":"slow down"}`: "slow down",
		`{}`:                      "{}",
		`{"detail":"blocked"}`:    `{"detail":"blocked"}`,
	} {
		var apiErr *APIError
		if !errors.As(serveBody(t, http.StatusBadGateway, body), &apiErr) || apiErr.Message != want {
			t.Errorf("body %s: got %+v, want message %q", body, apiErr, want)
		}
	}
}

// The API's text is printed straight to a terminal. A hostile or intercepted
// endpoint must not be able to drive it with escape sequences.
func TestClient_ServerTextCannotEscapeTheTerminal(t *testing.T) {
	body := `{"error":"x","code":"C\u001b[2J","message":"red\u001b[31m text\u0007 and \u009b31m more"}`
	var apiErr *APIError
	if !errors.As(serveBody(t, http.StatusConflict, body), &apiErr) {
		t.Fatal("no APIError")
	}
	for _, s := range []string{apiErr.Message, apiErr.Code} {
		for _, r := range s {
			if unicode.IsControl(r) {
				t.Fatalf("control character %U survived in %q", r, s)
			}
		}
	}
}

func TestClient_LongServerMessageIsCapped(t *testing.T) {
	long := strings.Repeat("a", 5000)
	var apiErr *APIError
	if !errors.As(serveBody(t, http.StatusConflict, `{"error":"x","message":"`+long+`"}`), &apiErr) {
		t.Fatal("no APIError")
	}
	if len(apiErr.Message) > maxServerMessage+len("... (truncated)") || !strings.HasSuffix(apiErr.Message, "... (truncated)") {
		t.Fatalf("message not capped: %d bytes", len(apiErr.Message))
	}
}

// Bidi overrides and line separators can reorder or fake lines on a terminal
// as surely as an escape sequence can.
func TestCleanServerText_DropsInvisibleFormattingAndLineBreaks(t *testing.T) {
	got := cleanServerText("safe‮denied⁦x⁩ next\nline\ttab", 200)
	if got != "safedeniedxnext line tab" {
		t.Fatalf("cleanServerText = %q", got)
	}
}

// Property: cleaned server text never carries a control, format or line
// separator character and never exceeds the cap, whatever came in.
func TestCleanServerText_Property(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		in := rapid.String().Draw(rt, "in")
		out := cleanServerText(in, 50)
		for _, r := range out {
			if unicode.IsControl(r) || unicode.In(r, unicode.Cf, unicode.Zl, unicode.Zp) {
				rt.Fatalf("unsafe character %U in %q", r, out)
			}
		}
		if n := len([]rune(strings.TrimSuffix(out, "... (truncated)"))); n > 50 {
			rt.Fatalf("output has %d runes, cap 50", n)
		}
	})
}

// Metamorphic partner of the property above: text that is already safe and
// under the cap comes back unchanged, so cleaning never eats a real message.
func TestCleanServerText_SafeTextIsUnchanged(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		in := rapid.StringMatching(`[ -~]{0,50}`).Draw(rt, "in")
		if out := cleanServerText(in, 50); out != in {
			rt.Fatalf("cleanServerText(%q) = %q, want it unchanged", in, out)
		}
	})
}

// safeText draws printable text from every script: letters, marks, digits,
// punctuation, symbols and spaces, none of which cleaning removes.
func safeText(rt *rapid.T, label string) string {
	return rapid.StringOfN(rapid.RuneFrom(nil, unicode.L, unicode.M, unicode.N, unicode.P, unicode.S, unicode.Zs), 1, 300, -1).Draw(rt, label)
}

// Property: a non-English message survives either envelope intact.
func TestParseAPIError_UnicodeMessagesRoundTrip(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		message := safeText(rt, "message")
		nested, err := json.Marshal(map[string]any{"error": map[string]string{"message": message, "code": "NOT_FOUND"}})
		if err != nil {
			rt.Fatalf("marshal: %v", err)
		}
		flat, err := json.Marshal(map[string]string{"error": "identity_conflict", "code": "IDENTITY_CONFLICT", "message": message})
		if err != nil {
			rt.Fatalf("marshal: %v", err)
		}
		for _, body := range [][]byte{nested, flat} {
			if got := parseAPIError(http.StatusConflict, body); got.Message != message {
				rt.Fatalf("message = %q, want %q (body %s)", got.Message, message, body)
			}
		}
	})
}

// JSON that is not an error envelope falls back to the raw body, cleaned.
func TestParseAPIError_NonEnvelopeBodies(t *testing.T) {
	for _, tc := range []struct{ body, want string }{
		{"", ""},
		{"null", "null"},
		{"[]", "[]"},
		{`"just a string"`, `"just a string"`},
		{`{"error":{}}`, `{"error":{}}`},
	} {
		got := parseAPIError(http.StatusInternalServerError, []byte(tc.body))
		if got.StatusCode != 500 || got.Code != "" || got.Message != tc.want {
			t.Errorf("body %q: got %d/%q/%q, want 500/\"\"/%q", tc.body, got.StatusCode, got.Code, got.Message, tc.want)
		}
	}
}

// What a command prints for the conflict: the status, the code and the reason,
// on one line, with no JSON.
func TestAPIError_IdentityConflictPrintsTheReason(t *testing.T) {
	body := `{"code":"IDENTITY_CONFLICT","email":"dana@example.com","error":"identity_conflict","message":"This email already belongs to a different sign-in identity in this organization."}`
	got := parseAPIError(http.StatusConflict, []byte(body)).Error()
	want := "API error (409/IDENTITY_CONFLICT): This email already belongs to a different sign-in identity in this organization."
	if got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

// Property: whatever the flat envelope carries, code and message round-trip
// into the APIError untouched.
func TestClient_FlatErrorEnvelopeProperty(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		code := rapid.StringMatching(`[A-Z_]{1,24}`).Draw(rt, "code")
		message := rapid.StringMatching(`[ -~]{1,300}`).Draw(rt, "message")
		raw, err := json.Marshal(map[string]string{"error": strings.ToLower(code), "code": code, "message": message})
		if err != nil {
			rt.Fatalf("marshal: %v", err)
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write(raw)
		}))
		defer server.Close()

		var apiErr *APIError
		if !errors.As(NewClient(server.URL).Get(context.Background(), "/x", nil), &apiErr) {
			rt.Fatalf("no APIError")
		}
		if apiErr.Code != code || apiErr.Message != message {
			rt.Fatalf("got %q/%q, want %q/%q", apiErr.Code, apiErr.Message, code, message)
		}
	})
}
