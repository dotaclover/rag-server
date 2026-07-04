package rag

import "strings"

// ScoreWeights 控制向量和关键词两项得分在最终相关性分数中的权重。
type ScoreWeights struct {
	Vector  float64 `json:"vector"`
	Keyword float64 `json:"keyword"`
}

// DefaultWeights 是两路混合评分的默认权重。
var DefaultWeights = ScoreWeights{Vector: 0.5, Keyword: 0.5}

// DomainProfile 携带领域特定的检索调优参数，由通用引擎在查询时应用。
// nil profile 表示纯通用行为：不扩展同义词、使用默认权重、不过滤来源。
type DomainProfile struct {
	Name     string              `json:"name"`
	Synonyms map[string][]string `json:"synonyms"`
	Phrases  []string            `json:"phrases,omitempty"`
	Weights  ScoreWeights        `json:"weights"`
	Sources  []string            `json:"sources"`
}

// weights 返回 profile 的权重；当 profile 为 nil 或权重全为零时回退到 DefaultWeights。
func (p *DomainProfile) weights() ScoreWeights {
	if p == nil {
		return DefaultWeights
	}
	if p.Weights.Vector == 0 && p.Weights.Keyword == 0 {
		return DefaultWeights
	}
	return p.Weights
}

// expand 在查询命中同义词键时将对应的扩展词追加到（已小写化的）查询中。
// 无 profile 或无同义词时原样返回。
func (p *DomainProfile) expand(query string) string {
	if p == nil || len(p.Synonyms) == 0 {
		return query
	}
	out := query
	for key, terms := range p.Synonyms {
		if strings.Contains(query, strings.ToLower(key)) {
			out += " " + strings.Join(terms, " ")
		}
	}
	return out
}

// allowsSource 检查文档来源是否通过 profile 的来源过滤。
// 空过滤器（或 nil profile）放行所有来源，因此单一合并索引仅在设置 Sources 后才有领域隔离效果。
func (p *DomainProfile) allowsSource(source string) bool {
	if p == nil || len(p.Sources) == 0 {
		return true
	}
	for _, s := range p.Sources {
		if s == source {
			return true
		}
	}
	return false
}
