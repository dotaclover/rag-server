package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"acflow-rag/rag"
)

const fallbackDoc = `# 劳动法知识样例

## 劳动合同

建立劳动关系，应当订立书面劳动合同。

## 试用期

劳动合同期限三年以上固定期限和无固定期限劳动合同，试用期不得超过六个月。

## 加班

休息日安排劳动者工作又不能安排补休的，应当支付不低于工资百分之二百的工资报酬。
`

func main() {
	engine := rag.NewEngine()
	engine.AddDocument("labor-law-sample", loadSampleDoc())

	mux := http.NewServeMux()
	mux.HandleFunc("/", home)
	mux.HandleFunc("/api/ask", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		if query == "" {
			query = "试用期最长多久？"
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(engine.Answer(query))
	})

	log.Println("acflow-rag listening on http://localhost:8082")
	log.Fatal(http.ListenAndServe(":8082", mux))
}

func loadSampleDoc() string {
	content, err := os.ReadFile("data/labor-law-sample.md")
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
  <input id="q" value="试用期最长多久？" />
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
