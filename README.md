# Dify 产品文档 RAG 检索服务

中文产品文档检索 Demo，数据源来自 Dify 官方文档仓库的中文文档精选页面。

## 数据源与许可

- 来源仓库：https://github.com/langgenius/dify-docs
- 许可协议：Creative Commons Attribution 4.0 International (CC-BY-4.0)
- 当前数据：约 410 条精选中文文档片段
- 说明文件：`data/NOTICE.md`

## 设计取舍：不是通用知识库，而是 Dify 领域 Demo

当前服务有意绑定 Dify 中文产品文档。`data/source.jsonl`、构建脚本、默认 profile、关键词和查询扩展里会出现 Dify、Dify Cloud、工作流、对话流、知识库、Docker Compose、Community Edition 等词。这是为了演示稳定性做出的选择，不是因为不知道这些逻辑可以配置化。

这样做的好处是：

- 演示主题集中，HR 或面试官能快速理解“产品手册问答”的业务场景。
- 针对 Dify 的同义词和短追问扩展可以提高召回效果，例如“本机 Win11 呢”“线上版本区别”这类问题。
- 数据源公开、中文可读、许可清晰，适合对外展示。

代价是：当前代码不是完全中立的通用 RAG 引擎。尤其是检索关键词、query expansion 和示例页面都偏向 Dify。

Go 代码中也有明确的 Dify hard code，需要被视为当前演示 profile 的一部分：

- `rag/search.go` 里的 `keywordTerms` 包含 Dify、workflow、chatflow、知识库、Docker Compose、Community Edition 等领域关键词。
- `rag/search.go` 里的 `expandProductDocsQuery` 会针对“线上 / 本地 / 安装 / 部署 / 主要功能 / 版本”等短问题追加 Dify 语境。
- `rag/profiles/dify.go` 提供内置 Dify profile，同义词和短语都围绕 Dify 产品文档。
- `tools/build_keyword_index.go` 里的默认模型标识是 `keyword+dify-docs-zh`。

这些 hard code 目前是有意保留的，因为它们能让演示问答更稳。如果要改成通用 RAG，建议先把 `keywordTerms`、`expandProductDocsQuery`、默认 model 名称和内置 profile 都迁移到 `profile.json` 或 per-domain 配置中。

如果要添加其他资料，有两种路径：

1. 继续作为 Dify 文档补充：把新资料转成同样的 JSONL 结构，合并到 `data/source.jsonl`，重新生成 `data/index.bin`。
2. 做成通用多知识库：新增 `data/domains/<domain>/source.jsonl`、`index.bin`、`profile.json`、`NOTICE.md`，把领域词表、同义词、来源过滤和展示文案放进 profile，再让 API 请求携带 `domain`。

推荐的长期结构：

```text
data/domains/
  dify_docs/
    source.jsonl
    index.bin
    profile.json
    NOTICE.md
  company_handbook/
    source.jsonl
    index.bin
    profile.json
    NOTICE.md
  product_manual/
    source.jsonl
    index.bin
    profile.json
    NOTICE.md
```

## 快速开始

```bash
go run main.go
```

访问：

```text
http://127.0.0.1:9093
```

## API

搜索：

```bash
curl -X POST http://localhost:9093/api/search \
  -H "Content-Type: application/json" \
  -d '{"query":"Dify 怎么创建知识库","top_k":5}'
```

响应示例：

```json
{
  "query": "Dify 怎么创建知识库",
  "results": [
    {
      "id": "dify_0001",
      "title": "创建知识库",
      "source": "Dify 中文文档",
      "section": "概览",
      "text": "知识库用于导入文档与数据...",
      "score": 0.85
    }
  ],
  "total": 5,
  "min_score": 0.5
}
```

最低相关度：

```bash
curl -X POST http://localhost:9093/api/search \
  -H "Content-Type: application/json" \
  -d '{"query":"Dify 应用如何发布","top_k":5,"min_score":60}'
```

`min_score` 支持 `60` 或 `0.6` 两种写法。没有传时使用服务配置，默认 50%。

状态：

```bash
curl http://localhost:9093/api/status
```

## 配置

```bash
go run main.go -host 127.0.0.1 -port 9093 -index data/index.bin -min-score 50
```

也可以用环境变量：

```bash
RAG_MIN_SCORE=50 go run main.go
```

## 重新生成数据

拉取 Dify 文档：

```bash
git clone --depth 1 https://github.com/langgenius/dify-docs.git /tmp/dify-docs
```

生成 JSONL：

```bash
node tools/build_dify_dataset.mjs /tmp/dify-docs
```

生成索引：

```bash
go run tools/build_keyword_index.go data/source.jsonl data/index.bin
```

## 示例问题

- Dify 里工作流和对话流有什么区别？
- Dify 怎么创建知识库？
- 知识库怎么测试检索效果？
- Dify 应用如何发布为 WebApp？
- 团队成员和模型供应商在哪里配置？

## 注意

- 当前主检索服务使用关键词匹配，`Embedding` 字段仅用于保持索引格式兼容。
- 如需真正向量检索，可接入 `embeding-server` 并使用 `rag/` 包里的向量检索实现。
- 公开展示时请保留 `data/NOTICE.md` 中的 Dify 文档署名和 CC-BY-4.0 许可说明。
