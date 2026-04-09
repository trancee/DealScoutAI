package jsonpath

import (
	"fmt"
	"strconv"
	"strings"
)

// Walk traverses a JSON structure using a dot-notation path.
// Supports map key access and array index access (e.g., "prices.0.value").
func Walk(data interface{}, path string) interface{} {
	if path == "" || data == nil {
		return data
	}

	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			current = v[part]
		case []interface{}:
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(v) {
				return nil
			}
			current = v[idx]
		default:
			return nil
		}
		if current == nil {
			return nil
		}
	}

	return current
}

// String extracts a string value at the given path.
func String(data interface{}, path string) string {
	if path == "" {
		return ""
	}
	val := Walk(data, path)
	if val == nil {
		return ""
	}
	return fmt.Sprintf("%v", val)
}

// Float extracts a float64 value at the given path.
func Float(data interface{}, path string) (float64, error) {
	if path == "" {
		return 0, fmt.Errorf("empty path")
	}
	val := Walk(data, path)
	if val == nil {
		return 0, fmt.Errorf("path %q resolved to nil", path)
	}

	switch v := val.(type) {
	case float64:
		return v, nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("unexpected type %T at path %q", val, path)
	}
}
