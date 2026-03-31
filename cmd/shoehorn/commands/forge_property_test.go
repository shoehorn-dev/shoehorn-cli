package commands

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
	"pgregory.net/rapid"
)

// Property: buildInputs with valid JSON always produces a superset of JSON keys.
func TestBuildInputs_Property_JSONKeysPreserved(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random map of string->string
		keys := rapid.SliceOfN(rapid.StringMatching(`^[a-z][a-z0-9_]{0,9}$`), 0, 5).Draw(t, "keys")
		m := map[string]string{}
		for _, k := range keys {
			m[k] = rapid.String().Draw(t, "val")
		}

		jsonBytes, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}

		result, err := buildInputs(string(jsonBytes), nil)
		if err != nil {
			t.Fatal(err)
		}

		// Property: every key from the JSON must be in the result
		for k := range m {
			if _, exists := result[k]; !exists {
				t.Fatalf("key %q from JSON missing in result", k)
			}
		}
	})
}

// Property: buildInputs with KV pairs always produces entries for every valid pair.
func TestBuildInputs_Property_KVPairsPreserved(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(0, 5).Draw(t, "n")
		pairs := make([]string, n)
		keys := make([]string, n)
		for i := range n {
			k := rapid.StringMatching(`^[a-z][a-z0-9_]{0,9}$`).Draw(t, "key")
			v := rapid.String().Draw(t, "val")
			pairs[i] = k + "=" + v
			keys[i] = k
		}

		result, err := buildInputs("", pairs)
		if err != nil {
			t.Fatal(err)
		}

		for _, k := range keys {
			if _, exists := result[k]; !exists {
				t.Fatalf("key %q from KV pairs missing in result", k)
			}
		}
	})
}

// Property: KV pairs override JSON values (last-write-wins for same key).
func TestBuildInputs_Property_KVOverridesJSON(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		key := rapid.StringMatching(`^[a-z]{1,5}$`).Draw(t, "key")
		jsonVal := rapid.String().Draw(t, "jsonVal")
		kvVal := rapid.String().Draw(t, "kvVal")

		jsonStr, _ := json.Marshal(map[string]string{key: jsonVal})
		result, err := buildInputs(string(jsonStr), []string{key + "=" + kvVal})
		if err != nil {
			t.Fatal(err)
		}

		if result[key] != kvVal {
			t.Fatalf("KV should override JSON: got %q, want %q", result[key], kvVal)
		}
	})
}

// Property: resolveAction with explicit flag always returns that flag value.
func TestResolveAction_Property_ExplicitFlagAlwaysWins(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		flag := rapid.StringMatching(`^[a-z]{1,10}$`).Draw(t, "flag")
		nActions := rapid.IntRange(0, 5).Draw(t, "nActions")
		actions := make([]api.MoldAction, nActions)
		for i := range nActions {
			actions[i] = api.MoldAction{
				Action:  rapid.StringMatching(`^[a-z]{1,10}$`).Draw(t, "action"),
				Primary: rapid.Bool().Draw(t, "primary"),
			}
		}

		got := resolveAction(flag, actions)
		if got != flag {
			t.Fatalf("resolveAction(%q, ...) = %q, want explicit flag", flag, got)
		}
	})
}

// Property: resolveAction with empty flag and non-empty actions never returns empty.
func TestResolveAction_Property_NonEmptyActionsNeverReturnsEmpty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		nActions := rapid.IntRange(1, 5).Draw(t, "nActions")
		actions := make([]api.MoldAction, nActions)
		for i := range nActions {
			actions[i] = api.MoldAction{
				Action:  rapid.StringMatching(`^[a-z]{1,10}$`).Draw(t, "action"),
				Primary: rapid.Bool().Draw(t, "primary"),
			}
		}

		got := resolveAction("", actions)
		if got == "" {
			t.Fatal("resolveAction(\"\", non-empty actions) returned empty string")
		}
	})
}

// Property: coerceInputTypes preserves non-string values (JSON-sourced).
func TestCoerceInputTypes_Property_NonStringsUntouched(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		boolVal := rapid.Bool().Draw(t, "boolVal")
		intVal := rapid.Int().Draw(t, "intVal")

		inputs := map[string]any{
			"flag":  boolVal,
			"count": intVal,
		}
		schema := []api.MoldInput{
			{Name: "flag", Type: "boolean"},
			{Name: "count", Type: "integer"},
		}

		coerceInputTypes(inputs, schema)

		// Property: values that were already the correct type should not change
		if inputs["flag"] != boolVal {
			t.Fatalf("bool value changed: got %v, want %v", inputs["flag"], boolVal)
		}
		if inputs["count"] != intVal {
			t.Fatalf("int value changed: got %v, want %v", inputs["count"], intVal)
		}
	})
}

// Property: coerceInputTypes with "true"/"false" strings always produces bool.
func TestCoerceInputTypes_Property_BoolStringsCoerced(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		boolStr := rapid.SampledFrom([]string{"true", "false", "1", "0", "TRUE", "FALSE"}).Draw(t, "boolStr")

		inputs := map[string]any{"flag": boolStr}
		schema := []api.MoldInput{{Name: "flag", Type: "boolean"}}

		coerceInputTypes(inputs, schema)

		if _, ok := inputs["flag"].(bool); !ok {
			t.Fatalf("expected bool after coercion of %q, got %T", boolStr, inputs["flag"])
		}
	})
}

// Property: truncateID output length is always <= 12.
func TestTruncateID_Property_MaxLength(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		id := rapid.String().Draw(t, "id")
		got := truncateID(id)
		if len(got) > 12 {
			t.Fatalf("truncateID(%q) returned %d chars, max 12", id, len(got))
		}
	})
}

// Property: truncateID is idempotent (truncating twice gives same result).
func TestTruncateID_Property_Idempotent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		id := rapid.String().Draw(t, "id")
		once := truncateID(id)
		twice := truncateID(once)
		if once != twice {
			t.Fatalf("not idempotent: truncateID(%q) = %q, truncateID(%q) = %q", id, once, once, twice)
		}
	})
}

// Property: NormalizeServerURL always produces a URL with a scheme.
func TestNormalizeServerURL_Property_AlwaysHasScheme(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		host := rapid.StringMatching(`^[a-z0-9][a-z0-9.-]{0,20}$`).Draw(t, "host")
		port := rapid.SampledFrom([]string{"", ":8080", ":443", ":3000"}).Draw(t, "port")
		scheme := rapid.SampledFrom([]string{"", "http://", "https://"}).Draw(t, "scheme")

		input := scheme + host + port
		got := NormalizeServerURL(input)

		if !strings.HasPrefix(got, "http://") && !strings.HasPrefix(got, "https://") {
			t.Fatalf("NormalizeServerURL(%q) = %q, missing scheme", input, got)
		}
	})
}

// Property: NormalizeServerURL never has trailing slash.
func TestNormalizeServerURL_Property_NoTrailingSlash(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		host := rapid.StringMatching(`^[a-z0-9][a-z0-9.-]{0,20}$`).Draw(t, "host")
		trailing := rapid.SampledFrom([]string{"", "/", "//", "///"}).Draw(t, "trailing")

		input := "http://" + host + trailing
		got := NormalizeServerURL(input)

		if strings.HasSuffix(got, "/") {
			t.Fatalf("NormalizeServerURL(%q) = %q, has trailing slash", input, got)
		}
	})
}
