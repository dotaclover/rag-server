# 多知识库 RAG 检索服务

这是一个单实例、多 domain 的中文 RAG Demo。服务启动后一次加载 `data/domains/*/index.bin`，API 通过 `domain` 参数选择知识库。

当前内置 domain：

| Domain | 数据源 | 用途 |
|---|---|---|
| `labor_law` | 中文劳动法/劳动合同法资料 | `rag.acfunc.online` 演示页默认知识库 |
| `dify_docs` | Dify 官方中文文档精选片段 | Agent 工具 `search_product_docs` 使用 |

两个 domain 都使用 `bge-small-zh-v1.5` 生成的 512 维向量索引，并通过各自 profile 调整关键词、同义词和混合检索权重。

## 目录结构

```text
data/domains/
  labor_law/
    source.jsonl
    index.bin
  dify_docs/
    source.jsonl
    index.bin
    NOTICE.md
```

Dify 文档来源：https://github.com/langgenius/dify-docs ，许可为 CC-BY-4.0，公开展示时请保留 `data/domains/dify_docs/NOTICE.md`。

## 本地启动

```bash
go run main.go -domains-dir data/domains -default-domain labor_law -min-score 0.3
```

或 Windows：

```bat
start.bat
```

默认访问：

```text
http://127.0.0.1:9093
```

## API

默认搜索劳动法：

```bash
curl -X POST http://127.0.0.1:9093/api/search \
  -H "Content-Type: application/json" \
  -d '{"query":"试用期一般多久","top_k":5}'
```

指定 Dify 文档：

```bash
curl -X POST 'http://127.0.0.1:9093/api/search?domain=dify_docs' \
  -H "Content-Type: application/json" \
  -d '{"query":"Dify 怎么创建知识库","top_k":5}'
```

也可以把 `domain` 放在 JSON body：

```json
{
  "domain": "dify_docs",
  "query": "Dify 里工作流和对话流有什么区别",
  "top_k": 5,
  "min_score": 0.3
}
```

`min_score` 支持 `0.5` 或 `50` 两种写法。

状态：

```bash
curl http://127.0.0.1:9093/api/status
curl http://127.0.0.1:9093/api/status?domain=labor_law
curl http://127.0.0.1:9093/api/status?domain=dify_docs
```

## Profile 说明

领域相关的 hard code 已集中到 `rag/profiles/`：

- `rag/profiles/labor.go`：劳动法关键词、同义词、混合检索权重。
- `rag/profiles/dify.go`：Dify 产品文档关键词、同义词、短问扩展和权重。

通用检索层 `rag/search.go` 不再直接绑定 Dify 或劳动法。当前仍保留内置 Go profile，是为了演示稳定；后续可以把这些配置迁移到每个 domain 的 `profile.json`。

## 重新生成索引

```bash
go run ./cmd/buildindex \
  -input data/domains/<domain>/source.jsonl \
  -output data/domains/<domain>/index.bin \
  -embedding-url http://127.0.0.1:9092
```
