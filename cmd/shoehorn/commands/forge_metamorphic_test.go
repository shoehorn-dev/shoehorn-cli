package commands

import (
	"encoding/json"
	"testing"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
	"pgregory.net/rapid"
)

// Metamorphic: buildInputs with empty JSON + empty KV produces empty map.
// Adding any input should make the result non-empty.
func TestBuildInputs_Metamorphic_EmptyIsIdentity(t *testing.T) {
	empty, err := buildInputs("", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("empty inputs produced %d keys, want 0", len(empty))
	}
}

// Metamorphic: buildInputs is additive -- adding more KV pairs only adds keys.
func TestBuildInputs_Metamorphic_Additive(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		base := rapid.SliceOfN(rapid.StringMatching(`^[a-z]{1,5}=[a-z]{1,5}$`), 0, 3).Draw(t, "base")
		extra := rapid.StringMatching(`^[a-z]{1,5}=[a-z]{1,5}$`).Draw(t, "extra")

		resultBase, err := buildInputs("", base)
		if err != nil {
			t.Fatal(err)
		}

		resultMore, err := buildInputs("", append(base, extra))
		if err != nil {
			t.Fatal(err)
		}

		// Property: result with more inputs has >= keys than result without
		if len(resultMore) < len(resultBase) {
			t.Fatalf("adding input reduced key count: %d -> %d", len(resultBase), len(resultMore))
		}
	})
}

// Metamorphic: JSON round-trip -- buildInputs with JSON serialized from a map
// should reconstruct at least the same keys.
func TestBuildInputs_Metamorphic_JSONRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(0, 5).Draw(t, "n")
		original := map[string]any{}
		for range n {
			k := rapid.StringMatching(`^[a-z]{1,5}$`).Draw(t, "key")
			v := rapid.StringMatching(`^[a-z]{1,10}$`).Draw(t, "val")
			original[k] = v
		}

		jsonBytes, _ := json.Marshal(original)
		result, err := buildInputs(string(jsonBytes), nil)
		if err != nil {
			t.Fatal(err)
		}

		for k, v := range original {
			if result[k] != v {
				t.Fatalf("round-trip lost key %q: got %v, want %v", k, result[k], v)
			}
		}
	})
}

// Metamorphic: resolveAction with a primary action always selects it
// regardless of action list order.
func TestResolveAction_Metamorphic_PrimaryOrderIndependent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		primary := rapid.StringMatching(`^[a-z]{1,8}$`).Draw(t, "primary")
		nOther := rapid.IntRange(0, 4).Draw(t, "nOther")
		others := make([]api.MoldAction, nOther)
		for i := range nOther {
			others[i] = api.MoldAction{
				Action:  rapid.StringMatching(`^[a-z]{1,8}$`).Draw(t, "other"),
				Primary: false,
			}
		}

		primaryAction := api.MoldAction{Action: primary, Primary: true}

		// Insert primary at random position
		pos := rapid.IntRange(0, len(others)).Draw(t, "pos")
		actions := make([]api.MoldAction, 0, len(others)+1)
		actions = append(actions, others[:pos]...)
		actions = append(actions, primaryAction)
		actions = append(actions, others[pos:]...)

		got := resolveAction("", actions)
		if got != primary {
			t.Fatalf("resolveAction with primary %q at position %d returned %q", primary, pos, got)
		}
	})
}

// Metamorphic: coerceInputTypes is idempotent -- running it twice gives same result.
func TestCoerceInputTypes_Metamorphic_Idempotent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		inputs := map[string]any{
			"flag":  rapid.SampledFrom([]string{"true", "false"}).Draw(t, "boolStr"),
			"count": rapid.SampledFrom([]string{"1", "42", "0"}).Draw(t, "intStr"),
		}
		schema := []api.MoldInput{
			{Name: "flag", Type: "boolean"},
			{Name: "count", Type: "integer"},
		}

		coerceInputTypes(inputs, schema)
		firstPass := map[string]any{}
		for k, v := range inputs {
			firstPass[k] = v
		}

		coerceInputTypes(inputs, schema)

		for k := range firstPass {
			if inputs[k] != firstPass[k] {
				t.Fatalf("coerceInputTypes not idempotent for key %q: %v -> %v", k, firstPass[k], inputs[k])
			}
		}
	})
}

// Metamorphic: formatStatus always contains the original status string.
func TestFormatStatus_Metamorphic_ContainsOriginal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		status := rapid.StringMatching(`^[a-z_]{1,15}$`).Draw(t, "status")
		result := formatStatus(status)
		if len(result) < len(status) {
			t.Fatalf("formatStatus(%q) = %q, shorter than input", status, result)
		}
		// The status string must appear in the result (after the icon prefix)
		found := false
		for i := 0; i <= len(result)-len(status); i++ {
			if result[i:i+len(status)] == status {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("formatStatus(%q) = %q, does not contain original status", status, result)
		}
	})
}
