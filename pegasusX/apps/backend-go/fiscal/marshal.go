package fiscal

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// MarshalCanonical produces deterministic JSON suitable for digital signatures.
// It recursively sorts all object keys.
func MarshalCanonical(v any) ([]byte, error) {
	// First convert to standard JSON bytes.
	// We use the standard library json.Marshal, which inherently handles
	// custom struct tags, omitempty, etc.
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("initial marshal failed: %w", err)
	}

	// Unmarshal back into a generic interface using UseNumber to preserve
	// exact precision of numbers, avoiding float64 conversions.
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	var generic any
	if err := dec.Decode(&generic); err != nil {
		return nil, fmt.Errorf("decode to generic failed: %w", err)
	}

	// Now re-marshal. Since `generic` is a map[string]any or []any or primitive,
	// json.Marshal will sort the keys of any map[string]any lexicographically.
	// It will also emit JSON with no insignificant whitespace.
	canonical, err := json.Marshal(generic)
	if err != nil {
		return nil, fmt.Errorf("canonical marshal failed: %w", err)
	}

	return canonical, nil
}
