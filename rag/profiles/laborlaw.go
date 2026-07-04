// Package profiles 存放通用 rag 引擎通过 rag.DomainProfile 消费的领域特定检索调优。
// 每个领域提供一个内置 Go profile，运行时可通过 data/domains/<domain>/profile.json 覆盖。
package profiles

import (
	"encoding/json"
	"os"
	"path/filepath"

	"go-agent-studio/services/rag"
)

// LaborLawProfile 是劳动法知识领域的内置检索调优参数。
// 同义词词典和短语列表逐字搬自引擎之前硬编码的 expandQuery/queryPhrases，保证召回不退化。
// Sources 暂为空（当前索引只有劳动法一个领域）；待第二个领域共享同一索引时再填入。
func LaborLawProfile() *rag.DomainProfile {
	return &rag.DomainProfile{
		Name: "labor_law",
		Synonyms: map[string][]string{
			"未签劳动合同": {"书面劳动合同", "一个月", "二倍工资", "劳动合同订立"},
			"没签劳动合同": {"书面劳动合同", "一个月", "二倍工资", "劳动合同订立"},
			"加班":     {"延长工作时间", "休息日", "法定休假日", "加班工资"},
			"违法解除":   {"解除劳动合同", "经济补偿", "赔偿金"},
			"工伤":     {"工伤保险", "工伤认定", "劳动能力鉴定"},
			"年假":     {"带薪年休假", "休息休假"},
			"试用期":    {"试用期间", "试用期内", "不得超过一个月", "不得超过二个月", "不得超过六个月", "同一用人单位", "不符合录用条件", "录用条件", "提前三日", "说明理由"},
			"试用期多久":  {"劳动合同期限三个月以上不满一年", "不得超过一个月", "一年以上不满三年", "不得超过二个月", "三年以上", "不得超过六个月"},
			"试用期多长":  {"劳动合同期限三个月以上不满一年", "不得超过一个月", "一年以上不满三年", "不得超过二个月", "三年以上", "不得超过六个月"},
			"提前多少天":  {"提前三日", "提前三十日", "书面形式通知", "额外支付劳动者一个月工资"},
			"多少天通知":  {"提前三日", "提前三十日", "书面形式通知"},
			"公司解除":   {"用人单位解除劳动合同", "用人单位可以解除劳动合同", "说明理由"},
			"公司需要":   {"用人单位", "通知劳动者本人", "说明理由"},
		},
		Phrases: []string{
			"试用期", "试用期间", "试用期内", "解除劳动合同", "解除合同",
			"劳动合同期限三个月以上不满一年", "不得超过一个月", "一年以上不满三年", "不得超过二个月", "三年以上", "不得超过六个月",
			"用人单位解除劳动合同", "用人单位可以解除劳动合同", "被证明不符合录用条件",
			"不符合录用条件", "录用条件", "提前三日", "提前3日", "提前三十日", "提前30日",
			"书面形式通知", "通知劳动者本人", "额外支付劳动者一个月工资", "说明理由",
		},
		Weights: rag.DefaultWeights,
	}
}

// profileFileName 是 loader 在领域索引同目录下查找的覆盖文件名。
const profileFileName = "profile.json"

// Load 返回某个领域的检索 profile。
// 优先读取 domainDir（即该领域索引所在目录）下的 profile.json，
// 以便无需重新编译即可调整检索调优；文件不存在或解析失败时回退到内置 profile。
func Load(domainDir, domain string, builtin *rag.DomainProfile) *rag.DomainProfile {
	path := filepath.Join(domainDir, profileFileName)
	raw, err := os.ReadFile(path)
	if err != nil {
		return builtin
	}
	var p rag.DomainProfile
	if err := json.Unmarshal(raw, &p); err != nil {
		return builtin
	}
	if p.Name == "" {
		p.Name = domain
	}
	if p.Weights == (rag.ScoreWeights{}) && builtin != nil {
		p.Weights = builtin.Weights
	}
	return &p
}
