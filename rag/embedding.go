package rag

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

const maxEmbeddingResponseBytes = 4 << 20

type Embedder interface {
	Name() string
	Dimension() int
	Embed(ctx context.Context, text string) ([]float64, error)
}

// RandomEmbedder 从文本种子生成确定性伪随机向量（本地无 API 演示用）。
// 不同文本产生不同向量，但无语义 —— 向量得分近乎噪声，
// 检索实际依赖 keyword 匹配和 DomainProfile 调优。
// 默认维度 2048，与常见 OpenAI 兼容 embedding 模型对齐。
type RandomEmbedder struct {
	dim int
}

func NewRandomEmbedder(dim int) *RandomEmbedder {
	if dim <= 0 {
		dim = 2048
	}
	return &RandomEmbedder{dim: dim}
}

func (e *RandomEmbedder) Name() string   { return "random-embedding-local" }
func (e *RandomEmbedder) Dimension() int { return e.dim }

func (e *RandomEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	// 确定性伪随机：用 SHA-256 生成的 seed 为每个维度生成 [-1, 1) 的值。
	seed := sha256.Sum256([]byte(text))
	rng := newXorshift64(seed)
	vec := make([]float64, e.dim)
	for i := range vec {
		vec[i] = rng.nextFloat()*2 - 1
	}
	normalize(vec)
	return vec, nil
}

// xorshift64 是轻量级确定性伪随机数生成器，用于 RandomEmbedder。
type xorshift64 struct {
	state uint64
}

func newXorshift64(seed [32]byte) *xorshift64 {
	s := binary.LittleEndian.Uint64(seed[:8])
	if s == 0 {
		s = 1 // xorshift 需要非零种子
	}
	return &xorshift64{state: s}
}

func (x *xorshift64) nextFloat() float64 {
	x.state ^= x.state << 13
	x.state ^= x.state >> 7
	x.state ^= x.state << 17
	const norm = 1.0 / float64(^uint64(0))
	return float64(x.state) * norm
}

type OpenAIEmbedder struct {
	apiKey     string
	baseURL    string
	model      string
	dimension  int
	path       string // 默认 /embeddings，Ark 用 /embeddings/multimodal
	httpClient *http.Client
}

func NewOpenAIEmbedder(apiKey, baseURL, model string, dimension int) *OpenAIEmbedder {
	return NewOpenAIEmbedderWithPath(apiKey, baseURL, model, dimension, "/embeddings")
}

func NewOpenAIEmbedderWithPath(apiKey, baseURL, model string, dimension int, path string) *OpenAIEmbedder {
	if path == "" {
		path = "/embeddings"
	}
	return &OpenAIEmbedder{
		apiKey:     apiKey,
		baseURL:    strings.TrimRight(baseURL, "/"),
		model:      model,
		dimension:  dimension,
		path:       path,
		httpClient: &http.Client{Timeout: 90 * time.Second},
	}
}

func (e *OpenAIEmbedder) Name() string   { return e.model }
func (e *OpenAIEmbedder) Dimension() int { return e.dimension }

func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	reqBody := map[string]interface{}{
		"model": e.model,
		"input": text,
	}
	if e.dimension > 0 {
		reqBody["dimensions"] = e.dimension
	}
	data, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+e.path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := readLimitedBody(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read embedding response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("embedding API returned status %d: %s", resp.StatusCode, string(body))
	}
	return parseEmbeddingResponse(body)
}

func readLimitedBody(r io.Reader) ([]byte, error) {
	return io.ReadAll(io.LimitReader(r, maxEmbeddingResponseBytes))
}

// parseEmbeddingResponse 解析 OpenAI 标准格式（data 为数组）和 Ark 格式（data 为单对象）。
func parseEmbeddingResponse(body []byte) ([]float64, error) {
	// 先尝试标准格式：{"data": [{"embedding": [...]}]}
	var arr struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &arr); err == nil && len(arr.Data) > 0 && len(arr.Data[0].Embedding) > 0 {
		return arr.Data[0].Embedding, nil
	}
	// 再尝试 Ark 格式：{"data": {"embedding": [...]}}
	var obj struct {
		Data struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &obj); err == nil && len(obj.Data.Embedding) > 0 {
		return obj.Data.Embedding, nil
	}
	return nil, fmt.Errorf("cannot parse embedding response")
}

func normalize(vec []float64) {
	var sum float64
	for _, v := range vec {
		sum += v * v
	}
	if sum == 0 {
		return
	}
	base := math.Sqrt(sum)
	for i := range vec {
		vec[i] /= base
	}
}
