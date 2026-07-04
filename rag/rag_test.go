package rag

import "testing"

func TestEngineSearchAndAnswer(t *testing.T) {
	engine := NewEngine()
	engine.AddDocument("sample", `# Sample

## 试用期

三年以上固定期限和无固定期限劳动合同，试用期不得超过六个月。

## 加班

休息日安排劳动者工作又不能安排补休的，应当支付不低于工资百分之二百的工资报酬。
`)

	hits := engine.Search("试用期最长多久", 2)
	if len(hits) == 0 {
		t.Fatal("expected at least one search hit")
	}
	result := engine.Answer("试用期最长多久")
	if result["answer"] == "" {
		t.Fatalf("expected answer, got %+v", result)
	}
	if result["citations"] == nil {
		t.Fatalf("expected citations, got %+v", result)
	}
}
