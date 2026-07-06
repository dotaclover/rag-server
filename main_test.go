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
			{ID: "low", Text: "Dify 模型 配置"},
			{ID: "high", Text: "Dify 工作流 对话流 区别"},
		},
	}

	results := search("Dify 工作流和对话流区别", 5, 50)
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1: %#v", len(results), results)
	}
	if results[0].ID != "high" {
		t.Fatalf("got result %q, want high", results[0].ID)
	}
}

func TestSearchExpandsDifyCloudSelfHostFollowUp(t *testing.T) {
	originalStore := store
	t.Cleanup(func() { store = originalStore })

	store = &Store{
		Docs: []Document{
			{ID: "env", Text: "本地安装的插件可用，浏览和自动升级不可用。"},
			{ID: "home", Title: "Dify 文档", Text: "Dify Cloud 是托管平台，无需安装，并包含免费的 Sandbox 套餐。自部署是在自己的基础设施上运行开源的 Community Edition，使用 Docker Compose 几分钟即可完成部署。"},
		},
	}

	results := search("本地安装和线上版本有什么区别", 5, 50)
	if len(results) == 0 {
		t.Fatal("got no results")
	}
	if results[0].ID != "home" {
		t.Fatalf("got top result %q, want home: %#v", results[0].ID, results)
	}
}

func TestNoResultsMessageIncludesThreshold(t *testing.T) {
	got := noResultsMessage(0.5)
	if !strings.Contains(got, "50%") {
		t.Fatalf("message missing threshold: %s", got)
	}
}
