import fs from "node:fs";
import path from "node:path";

const root = process.argv[2] || process.env.DIFY_DOCS_DIR;
if (!root) {
  console.error("usage: node tools/build_dify_dataset.mjs <dify-docs-dir>");
  process.exit(1);
}

const selected = [
  "zh/home.mdx",
  "zh/quick-start.mdx",
  "zh/learn/key-concepts.mdx",
  "zh/self-host/deploy/overview.mdx",
  "zh/self-host/deploy/quick-start/docker-compose.mdx",
  "zh/self-host/deploy/quick-start/faqs.mdx",
  "zh/self-host/deploy/platform-guides/bt-panel.mdx",
  "zh/self-host/deploy/configuration/environments.mdx",
  "zh/self-host/use-dify/getting-started/introduction.mdx",
  "zh/cloud/use-dify/getting-started/introduction.mdx",
  "zh/cloud/use-dify/build/chatbot.mdx",
  "zh/cloud/use-dify/build/agent.mdx",
  "zh/cloud/use-dify/build/workflow-chatflow.mdx",
  "zh/cloud/use-dify/build/text-generator.mdx",
  "zh/cloud/use-dify/build/additional-features.mdx",
  "zh/cloud/use-dify/build/version-control.mdx",
  "zh/cloud/use-dify/knowledge/readme.mdx",
  "zh/cloud/use-dify/knowledge/create-knowledge/introduction.mdx",
  "zh/cloud/use-dify/knowledge/create-knowledge/setting-indexing-methods.mdx",
  "zh/cloud/use-dify/knowledge/create-knowledge/chunking-and-cleaning-text.mdx",
  "zh/cloud/use-dify/knowledge/create-knowledge/import-text-data/readme.mdx",
  "zh/cloud/use-dify/knowledge/test-retrieval.mdx",
  "zh/cloud/use-dify/knowledge/integrate-knowledge-within-application.mdx",
  "zh/cloud/use-dify/knowledge/manage-knowledge/introduction.mdx",
  "zh/cloud/use-dify/knowledge/manage-knowledge/maintain-knowledge-documents.mdx",
  "zh/cloud/use-dify/knowledge/metadata.mdx",
  "zh/cloud/use-dify/nodes/knowledge-retrieval.mdx",
  "zh/cloud/use-dify/nodes/llm.mdx",
  "zh/cloud/use-dify/nodes/answer.mdx",
  "zh/cloud/use-dify/nodes/start.mdx",
  "zh/cloud/use-dify/nodes/user-input.mdx",
  "zh/cloud/use-dify/nodes/ifelse.mdx",
  "zh/cloud/use-dify/nodes/code.mdx",
  "zh/cloud/use-dify/nodes/http-request.mdx",
  "zh/cloud/use-dify/nodes/question-classifier.mdx",
  "zh/cloud/use-dify/nodes/tools.mdx",
  "zh/cloud/use-dify/nodes/agent.mdx",
  "zh/cloud/use-dify/publish/README.mdx",
  "zh/cloud/use-dify/publish/developing-with-apis.mdx",
  "zh/cloud/use-dify/publish/webapp/web-app-settings.mdx",
  "zh/cloud/use-dify/publish/webapp/web-app-access.mdx",
  "zh/cloud/use-dify/publish/webapp/embedding-in-websites.mdx",
  "zh/cloud/use-dify/monitor/logs.mdx",
  "zh/cloud/use-dify/monitor/analysis.mdx",
  "zh/cloud/use-dify/workspace/team-members-management.mdx",
  "zh/cloud/use-dify/workspace/model-providers.mdx",
  "zh/cloud/use-dify/workspace/app-management.mdx",
  "zh/cloud/use-dify/workspace/plugins.mdx",
  "zh/cloud/use-dify/workspace/tools.mdx",
];

const outDir = path.resolve("data");
const outPath = path.join(outDir, "source.jsonl");
const noticePath = path.join(outDir, "NOTICE.md");
fs.mkdirSync(outDir, { recursive: true });

function parseFrontmatter(raw) {
  const match = raw.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n?/);
  const meta = {};
  if (!match) return { meta, body: raw };
  for (const line of match[1].split(/\r?\n/)) {
    const m = line.match(/^([A-Za-z0-9_-]+):\s*(.*)$/);
    if (!m) continue;
    meta[m[1]] = m[2].replace(/^["']|["']$/g, "").trim();
  }
  return { meta, body: raw.slice(match[0].length) };
}

function cleanMarkdown(text) {
  return text
    .replace(/\r\n/g, "\n")
    .replace(/```[\s\S]*?```/g, " ")
    .replace(/`([^`]+)`/g, "$1")
    .replace(/!\[[^\]]*\]\([^)]+\)/g, " ")
    .replace(/\[([^\]]+)\]\([^)]+\)/g, "$1")
    .replace(/^\s*import\s+.*$/gm, " ")
    .replace(/^\s*export\s+.*$/gm, " ")
    .replace(/<[^>\n]+>/g, " ")
    .replace(/^\s*\|?\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)+\|?\s*$/gm, " ")
    .replace(/[ \t]+/g, " ")
    .replace(/\n{3,}/g, "\n\n")
    .trim();
}

function splitSections(body, fallbackSection) {
  const lines = body.split("\n");
  const sections = [];
  let current = { section: fallbackSection, lines: [] };
  for (const line of lines) {
    const heading = line.match(/^(#{1,4})\s+(.+?)\s*$/);
    if (heading) {
      if (current.lines.join("\n").trim()) sections.push(current);
      current = { section: cleanInline(heading[2]), lines: [] };
      continue;
    }
    current.lines.push(line);
  }
  if (current.lines.join("\n").trim()) sections.push(current);
  return sections;
}

function cleanInline(text) {
  return text
    .replace(/<[^>]+>/g, "")
    .replace(/\[([^\]]+)\]\([^)]+\)/g, "$1")
    .replace(/`([^`]+)`/g, "$1")
    .replace(/\s+/g, " ")
    .trim();
}

function chunkText(text, maxChars = 760) {
  const paragraphs = text.split(/\n\s*\n/).map((p) => p.trim()).filter(Boolean);
  const chunks = [];
  let current = "";
  for (const p of paragraphs) {
    if ((current + "\n" + p).trim().length <= maxChars) {
      current = (current + "\n" + p).trim();
      continue;
    }
    if (current) chunks.push(current);
    if (p.length <= maxChars) {
      current = p;
      continue;
    }
    for (const part of splitLong(p, maxChars)) chunks.push(part);
    current = "";
  }
  if (current) chunks.push(current);
  return chunks.filter((chunk) => chunk.length >= 50);
}

function splitLong(text, maxChars) {
  const parts = [];
  let rest = text.trim();
  while (rest.length > maxChars) {
    let cut = rest.slice(0, maxChars).search(/[。！？；]\s*[^。！？；]*$/);
    if (cut < Math.floor(maxChars * 0.45)) cut = maxChars;
    const part = rest.slice(0, cut).trim();
    if (part) parts.push(part);
    rest = rest.slice(cut).trim();
  }
  if (rest) parts.push(rest);
  return parts;
}

const records = [];
for (const rel of selected) {
  const file = path.join(root, rel);
  if (!fs.existsSync(file)) {
    console.warn(`skip missing ${rel}`);
    continue;
  }
  const raw = fs.readFileSync(file, "utf8");
  const { meta, body } = parseFrontmatter(raw);
  const title = meta.title || meta.sidebarTitle || path.basename(rel, path.extname(rel));
  for (const section of splitSections(body, meta.description || "概览")) {
    const cleaned = cleanMarkdown(section.lines.join("\n"));
    for (const chunk of chunkText(cleaned)) {
      records.push({
        id: `dify_${String(records.length + 1).padStart(4, "0")}`,
        title,
        source: "Dify 中文文档",
        section: section.section || "概览",
        text: chunk,
        metadata: {
          path: rel.replace(/\\/g, "/"),
          license: "CC-BY-4.0",
          attribution: "Dify Documentation by LangGenius / Dify",
          source_url: `https://github.com/langgenius/dify-docs/blob/main/${rel.replace(/\\/g, "/")}`,
        },
      });
    }
  }
}

fs.writeFileSync(outPath, records.map((record) => JSON.stringify(record)).join("\n") + "\n", "utf8");
fs.writeFileSync(
  noticePath,
  `# Dify 中文文档数据源说明\n\n` +
    `本 RAG 示例数据来源于 Dify 官方文档仓库的中文文档精选页面。\n\n` +
    `- 来源仓库：https://github.com/langgenius/dify-docs\n` +
    `- 许可协议：Creative Commons Attribution 4.0 International (CC-BY-4.0)\n` +
    `- License 文件：https://github.com/langgenius/dify-docs/blob/main/LICENSE\n` +
    `- 内容作者：LangGenius / Dify Documentation contributors\n\n` +
    `本仓库将原始 MDX 文档清洗并切分为 JSONL 检索片段，用于中文产品文档问答 Demo。使用和分发时请保留以上署名和许可说明。\n`,
  "utf8",
);

console.log(`wrote ${records.length} records to ${outPath}`);
