# Dify 产品文档 RAG 检索服务

中文产品文档检索 Demo，数据源来自 Dify 官方文档仓库的中文文档精选页面。

## 数据源与许可

- 来源仓库：https://github.com/langgenius/dify-docs
- 许可协议：Creative Commons Attribution 4.0 International (CC-BY-4.0)
- 当前数据：272 条精选中文文档片段
- 说明文件：`data/NOTICE.md`

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
