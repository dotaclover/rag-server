package main

import (
	"strings"
	"testing"
)

func TestNormalizeScoreThreshold(t *testing.T) {
	tests := []struct {
		name string
		in   float64
		want float64
	}{
		{name: "percent", in: 50, want: 0.5},
		{name: "ratio", in: 0.5, want: 0.5},
		{name: "negative", in: -1, want: 0},
		{name: "over one hundred percent", in: 150, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeScoreThreshold(tt.in)
			if got != tt.want {
				t.Fatalf("normalizeScoreThreshold(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestSearchFiltersBelowMinScore(t *testing.T) {
	originalStore := store
	t.Cleanup(func() { store = originalStore })

	store = &Store{
		Docs: []Document{
			{ID: "low", Text: "试用期 工资"},
			{ID: "high", Text: "试用期 一般 多久"},
		},
	}

	results := search("试用期一般多久", 5, 50)
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1: %#v", len(results), results)
	}
	if results[0].ID != "high" {
		t.Fatalf("got result %q, want high", results[0].ID)
	}
}

func TestNoResultsMessageIncludesThreshold(t *testing.T) {
	got := noResultsMessage(0.5)
	if !strings.Contains(got, "50%") {
		t.Fatalf("message missing threshold: %s", got)
	}
}
