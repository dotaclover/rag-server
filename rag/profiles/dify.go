// Package profiles 存放通用 rag 引擎通过 rag.DomainProfile 消费的领域特定检索调优。
// 每个领域提供一个内置 Go profile，运行时可通过 data/domains/<domain>/profile.json 覆盖。
package profiles

import (
	"encoding/json"
	"os"
	"path/filepath"

	"acflow-rag/rag"
)

// DifyDocsProfile returns the default retrieval tuning for Dify product documentation.
func DifyDocsProfile() *rag.DomainProfile {
	return &rag.DomainProfile{
		Name: "dify_docs",
		Synonyms: map[string][]string{
			"工作流":  {"workflow", "单轮任务", "批量执行", "开始节点"},
			"对话流":  {"chatflow", "多轮对话", "会话变量", "记忆"},
			"知识库":  {"knowledge", "dataset", "文档", "检索", "分段", "索引"},
			"发布":   {"webapp", "api", "嵌入网站", "发布更新"},
			"模型":   {"模型供应商", "provider", "默认模型", "api key"},
			"安装":   {"部署", "自部署", "docker compose", "community edition", "dify cloud"},
			"本地":   {"自部署", "docker compose", "community edition", "线上", "dify cloud"},
			"功能":   {"应用类型", "工作流", "对话流", "知识库", "agent", "发布"},
			"团队":   {"成员", "权限", "workspace", "工作空间"},
			"节点":   {"llm", "answer", "ifelse", "http request", "code"},
			"测试检索": {"召回", "检索效果", "命中", "相似度"},
		},
		Expanders: map[string][]string{
			"线上":    {"dify", "dify cloud", "托管", "无需安装", "sandbox"},
			"cloud": {"dify", "dify cloud", "托管", "无需安装", "sandbox"},
			"本地":    {"dify", "自部署", "community edition", "docker compose", "基础设施"},
			"自部署":   {"dify", "自部署", "community edition", "docker compose", "基础设施"},
			"安装":    {"dify", "自部署", "docker compose", "community edition"},
			"部署":    {"dify", "自部署", "docker compose", "community edition"},
			"主要功能":  {"dify", "应用", "工作流", "对话流", "知识库", "agent", "发布"},
			"功能":    {"dify", "应用", "工作流", "对话流", "知识库", "agent", "发布"},
			"版本":    {"dify", "dify cloud", "自部署", "community edition"},
		},
		Phrases: []string{
			"Dify", "工作流", "对话流", "聊天助手", "Agent", "知识库", "知识检索",
			"文档分段", "索引方式", "测试检索", "模型供应商", "发布应用",
			"WebApp", "API", "监控日志", "团队成员", "插件", "节点",
			"Dify Cloud", "自部署", "Docker Compose", "Community Edition", "主要功能",
		},
		KeywordTerms: []string{
			"default model", "http request", "knowledge", "marketplace", "provider", "workflow", "chatflow",
			"embedding", "webapp", "agent", "dify", "dify cloud", "api", "llm", "mcp", "rag",
			"docker", "docker compose", "compose", "cloud", "sandbox", "community edition",
			"模型供应商", "外部知识库", "默认模型", "工作空间", "聊天助手", "文本生成", "问题分类器",
			"知识库", "工作流", "对话流", "数据源", "提示词", "发布", "应用", "节点", "模型",
			"插件", "集成", "工具", "团队", "成员", "权限", "变量", "会话", "记忆", "日志",
			"监控", "文档", "分段", "索引", "检索", "召回", "重排序", "嵌入", "接口", "密钥",
			"创建", "测试", "配置", "导入", "上传", "部署", "安装", "调用", "发布", "调试",
			"运行", "输出", "输入", "文件", "套餐", "用量", "主要功能", "功能", "区别",
			"本地", "线上", "版本", "自部署", "本地部署", "托管", "基础设施", "开源",
			"开箱即用", "沙箱", "社区版",
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
