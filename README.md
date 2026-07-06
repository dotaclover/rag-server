# ACFlow RAG 检索服务

劳动法智能检索系统 - 基于 BGE Embedding 的语义搜索

## 🚀 快速开始

```bash
# 运行服务
go run main.go

# 或使用启动脚本
start.bat

# 或运行编译后的可执行文件
./rag-server.exe
```

访问: **http://127.0.0.1:9093**

## ✨ 功能特性

- 🔍 **智能搜索** - 基于关键词匹配的搜索算法
- 🎨 **美观界面** - 现代化的渐变色设计
- 📊 **Top K 控制** - 可自定义返回结果数量（1-10）
- 📚 **524条文档** - 完整的劳动法知识库
- ⚡ **快速响应** - 纯 Go 实现，毫秒级响应

## 🎨 界面预览

- 渐变背景（紫色主题）
- 搜索框 + Top K 控制
- 示例查询快捷按钮
- 卡片式结果展示
- 相关度评分显示

## 📡 API 文档

### 搜索接口

```bash
POST /api/search
Content-Type: application/json

{
  "query": "试用期一般多久",
  "top_k": 5
}
```

响应:
```json
{
  "query": "试用期一般多久",
  "top_k": 5,
  "min_score": 50
}
```

`min_score` 可选，支持 `50` 或 `0.5` 两种写法，表示最低相关度 50%。未传时使用服务配置，默认 50%。

响应:
```json
{
  "query": "试用期一般多久",
  "results": [
    {
      "id": "labor_law_19",
      "title": "劳动合同法",
      "source": "劳动合同法",
      "section": "第十九条",
      "text": "试用期最长不得超过六个月...",
      "score": 0.95
    }
  ],
  "total": 5,
  "min_score": 0.5
}
```

没有达到阈值的结果时:

```json
{
  "query": "通用问题",
  "results": [],
  "total": 0,
  "min_score": 0.5,
  "message": "未找到相关度不低于 50% 的参考资料，请换个更具体的问题或降低阈值后重试。"
}
```

### 状态检查

```bash
GET /api/status
```

响应:
```json
{
  "loaded": true,
  "documents": 524,
  "dimension": 512,
  "model": "bge-small-zh-v1.5",
  "indexPath": "data/index.bin"
}
```

## 💡 示例查询

1. "试用期一般多久"
2. "加班工资怎么算"
3. "公司可以随意辞退员工吗"
4. "年假有几天"

## 🔧 配置选项

```bash
# 指定端口
go run main.go -port 9093

# 指定索引路径
go run main.go -index data/index.bin

# 指定最低相关度；50 和 0.5 都表示 50%
go run main.go -min-score 50

# 也可用环境变量配置
$env:RAG_MIN_SCORE='50'
go run main.go
```

## 📂 项目结构

```
acflow-rag/
├── main.go              # 主程序
├── templates/
│   └── index.html       # Web界面
├── static/
│   └── .gitkeep         # 静态文件目录
├── data/
│   ├── index.bin        # RAG索引（524文档）
│   └── source.jsonl     # 原始文档
├── go.mod
├── start.bat            # 启动脚本
└── README.md
```

## 🔍 搜索算法

当前使用关键词匹配算法：

```go
score = (matched_tokens / total_tokens) + phrase_bonus
```

- 匹配的关键词越多，分数越高
- 完全匹配查询短语会获得额外加分（+0.3）
- 分数范围：0.0 - 1.0

## 🚀 性能

- 启动时间: < 1秒
- 搜索响应: < 50ms
- 内存占用: ~10MB
- 索引大小: 1.7MB

## 📝 数据说明

- **文档数量**: 524条
- **数据来源**: 劳动法相关法规
- **Embedding**: BGE-small-zh-v1.5 (512维)
- **格式**: Go gob 二进制

## 🔗 集成示例

### cURL

```bash
curl -X POST http://localhost:9093/api/search \
  -H "Content-Type: application/json" \
  -d '{"query": "试用期", "top_k": 3}'
```

### Python

```python
import requests

response = requests.post(
    'http://localhost:9093/api/search',
    json={'query': '试用期', 'top_k': 5}
)
print(response.json())
```

### JavaScript

```javascript
fetch('http://localhost:9093/api/search', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({query: '试用期', top_k: 5})
})
.then(r => r.json())
.then(data => console.log(data));
```

## 🎯 使用场景

1. **法律咨询** - 劳动法问题快速检索
2. **学习工具** - RAG技术学习参考
3. **API服务** - 作为检索服务集成到其他应用
4. **Demo展示** - 向量检索技术演示

## ⚠️ 注意事项

- 当前使用关键词匹配，未使用向量相似度
- 如需向量检索，需集成 acflow-embeding 服务
- 索引文件必须存在：`data/index.bin`

## 📈 改进方向

1. 集成 acflow-embeding 实现真正的向量检索
2. 添加混合评分（向量 + 关键词）
3. 支持高级过滤（按来源、章节）
4. 添加搜索历史记录
5. 支持结果导出

## 📄 License

MIT License
