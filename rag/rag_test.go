package rag

import "testing"

func TestEngineSearchAndAnswer(t *testing.T) {
	engine := NewEngine()
	engine.AddDocument("sample", `# Sample

## 工作流

工作流适合处理单轮任务，可以通过 Web 应用界面和 API 批量执行。

## 知识库

知识库用于导入文档与数据，应用可以检索并引用这些内容。
`)

	hits := engine.Search("工作流适合什么任务", 2)
	if len(hits) == 0 {
		t.Fatal("expected at least one search hit")
	}
	result := engine.Answer("工作流适合什么任务")
	if result["answer"] == "" {
		t.Fatalf("expected answer, got %+v", result)
	}
	if result["citations"] == nil {
		t.Fatalf("expected citations, got %+v", result)
	}
}
