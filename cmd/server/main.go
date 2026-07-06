package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"acflow-rag/rag"
)

const fallbackDoc = `# Dify 产品文档知识样例

## 工作流

工作流适合处理单轮任务，可以通过 Web 应用界面和 API 批量执行。

## 对话流

对话流适合需要多轮上下文的对话场景，可以结合会话变量和记忆。

## 知识库

知识库用于导入文档与数据，应用可以检索并引用这些内容。
`

func main() {
	engine := rag.NewEngine()
	engine.AddDocument("dify-docs-sample", loadSampleDoc())

	mux := http.NewServeMux()
	mux.HandleFunc("/", home)
	mux.HandleFunc("/api/ask", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		if query == "" {
			query = "工作流适合什么任务？"
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(engine.Answer(query))
	})

	log.Println("acflow-rag listening on http://localhost:8082")
	log.Fatal(http.ListenAndServe(":8082", mux))
}

func loadSampleDoc() string {
	content, err := os.ReadFile("data/product-docs-sample.md")
	if err != nil {
		return fallbackDoc
	}
	return string(content)
}

func home(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>acflow-rag</title>
  <style>
    body{font-family:Arial,"Microsoft YaHei",sans-serif;margin:0;background:#f7f8fb;color:#1f2937}
    main{max-width:880px;margin:40px auto;padding:0 20px}
    input{width:100%;padding:12px;border:1px solid #cfd6e3;border-radius:6px;font:16px/1.5 inherit}
    button{margin-top:12px;padding:10px 14px;border:0;border-radius:6px;background:#0f766e;color:white;font-weight:700;cursor:pointer}
    pre{white-space:pre-wrap;background:white;border:1px solid #dde3ee;border-radius:6px;padding:16px;min-height:220px}
  </style>
</head>
<body>
<main>
  <h1>acflow-rag</h1>
  <p>最小 RAG Pipeline：Document -> Chunk -> Retrieve -> Prompt -> Answer -> Citation。</p>
  <input id="q" value="工作流适合什么任务？" />
  <button id="ask">提问</button>
  <pre id="out"></pre>
</main>
<script>
ask.onclick = async () => {
  out.textContent = "Retrieving...";
  const res = await fetch("/api/ask?q=" + encodeURIComponent(q.value));
  out.textContent = JSON.stringify(await res.json(), null, 2);
};
</script>
</body>
</html>`)
}
