package jsonpath_test

import (
	"testing"

	"github.com/trancee/DealScout/internal/jsonpath"
)

func TestWalk(t *testing.T) {
	data := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": "deep",
			},
		},
		"arr": []interface{}{
			map[string]interface{}{"x": 1.0},
			map[string]interface{}{"x": 2.0},
		},
		"top": "level",
	}

	tests := []struct {
		path string
		want interface{}
	}{
		{"top", "level"},
		{"a.b.c", "deep"},
		{"arr.0.x", 1.0},
		{"arr.1.x", 2.0},
		{"arr.2.x", nil},
		{"missing", nil},
		{"a.b.missing", nil},
		{"", data},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := jsonpath.Walk(data, tt.path)
			if tt.path == "" {
				if got == nil {
					t.Error("Walk(\"\") should return data, got nil")
				}
			} else if got != tt.want {
				t.Errorf("Walk(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestString(t *testing.T) {
	data := map[string]interface{}{"name": "test", "num": 42.0}

	if got := jsonpath.String(data, "name"); got != "test" {
		t.Errorf("String(name) = %q, want %q", got, "test")
	}
	if got := jsonpath.String(data, "num"); got != "42" {
		t.Errorf("String(num) = %q, want %q", got, "42")
	}
	if got := jsonpath.String(data, "missing"); got != "" {
		t.Errorf("String(missing) = %q, want empty", got)
	}
}

func TestFloat(t *testing.T) {
	data := map[string]interface{}{"price": 99.9, "str": "42.5"}

	if got, err := jsonpath.Float(data, "price"); err != nil || got != 99.9 {
		t.Errorf("Float(price) = %f, %v", got, err)
	}
	if got, err := jsonpath.Float(data, "str"); err != nil || got != 42.5 {
		t.Errorf("Float(str) = %f, %v", got, err)
	}
	if _, err := jsonpath.Float(data, "missing"); err == nil {
		t.Error("Float(missing) should error")
	}
}
