package main

import (
	"math"
	"testing"

	"acflow-rag/rag"
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
		{name: "nan fallback", in: math.NaN(), want: 0.5},
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

func TestDetectProfile(t *testing.T) {
	tests := []struct {
		name      string
		indexPath string
		store     *rag.Store
		want      string
	}{
		{
			name:      "labor index path",
			indexPath: "data/domains/labor_law/index.bin",
			store:     &rag.Store{},
			want:      "labor_law",
		},
		{
			name:      "labor source",
			indexPath: "data/index.bin",
			store: &rag.Store{Docs: []rag.Document{
				{Source: "劳动合同法"},
			}},
			want: "labor_law",
		},
		{
			name:      "dify source",
			indexPath: "data/index.bin",
			store: &rag.Store{Docs: []rag.Document{
				{Source: "Dify 中文文档"},
			}},
			want: "dify_docs",
		},
		{
			name:      "unknown",
			indexPath: "data/other.bin",
			store:     &rag.Store{},
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectProfile("", tt.indexPath, tt.store)
			if tt.want == "" {
				if got != nil {
					t.Fatalf("detectProfile() = %q, want nil", got.Name)
				}
				return
			}
			if got == nil || got.Name != tt.want {
				name := "<nil>"
				if got != nil {
					name = got.Name
				}
				t.Fatalf("detectProfile() = %q, want %q", name, tt.want)
			}
		})
	}
}

func TestDetectProfileUsesDomainNameFirst(t *testing.T) {
	got := detectProfile("dify_docs", "data/domains/labor_law/index.bin", &rag.Store{})
	if got == nil || got.Name != "dify_docs" {
		t.Fatalf("detectProfile() = %#v, want dify_docs", got)
	}
}

func TestNormalizeDomain(t *testing.T) {
	if got := normalizeDomain(" DIFY_DOCS "); got != "dify_docs" {
		t.Fatalf("normalizeDomain() = %q", got)
	}
}
