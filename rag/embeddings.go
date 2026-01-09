package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sashabaranov/go-openai"
)

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
	Dimension() int
}

// OpenAI Embedder (garde pour backup)
type OpenAIEmbedder struct {
	client *openai.Client
	model  string
	dim    int
}

func NewOpenAIEmbedder(model, apiKey string, dim int) (Embedder, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key required")
	}

	client := openai.NewClient(apiKey)
	return &OpenAIEmbedder{
		client: client,
		model:  model,
		dim:    dim,
	}, nil
}

func (e *OpenAIEmbedder) Dimension() int {
	return e.dim
}

func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	resp, err := e.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Model: openai.EmbeddingModel(e.model),
		Input: []string{text},
	})
	if err != nil {
		return nil, err
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return resp.Data[0].Embedding, nil
}

func (e *OpenAIEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	resp, err := e.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Model: openai.EmbeddingModel(e.model),
		Input: texts,
	})
	if err != nil {
		return nil, err
	}

	embeddings := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		embeddings[i] = data.Embedding
	}

	return embeddings, nil
}

// LM Studio Local Embedder
type LocalEmbedder struct {
	baseURL       string
	model         string
	dim           int
	httpClient    *http.Client
	maxBatchSize  int // Maximum number of texts per batch
	maxTokensHint int // Estimated max tokens per batch (rough approximation)
}

type EmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type EmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

func NewLocalEmbedder(baseURL, model string, dim int) (Embedder, error) {
	if baseURL == "" {
		baseURL = "http://localhost:1234/v1"
	}

	embedder := &LocalEmbedder{
		baseURL:       baseURL,
		model:         model,
		dim:           dim,
		maxBatchSize:  20,    // Aggressive: max 20 chunks per API call (20 × 1,500 tokens ≈ 30k)
		maxTokensHint: 28000, // Target ~28k tokens per batch (safe margin under 32k limit)
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Increased timeout for larger batches
		},
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := embedder.testConnection(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to LM Studio: %w", err)
	}

	return embedder, nil
}

func (e *LocalEmbedder) Dimension() int {
	return e.dim
}

func (e *LocalEmbedder) testConnection(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", e.baseURL+"/models", nil)
	if err != nil {
		return err
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("LM Studio returned status %d", resp.StatusCode)
	}

	return nil
}

func (e *LocalEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return embeddings[0], nil
}

func (e *LocalEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	// If batch is small enough, process directly
	if len(texts) <= e.maxBatchSize {
		estimatedTokens := e.estimateTokens(texts)
		if estimatedTokens <= e.maxTokensHint {
			return e.embedBatchDirect(ctx, texts)
		}
	}

	// Otherwise, split into smaller sub-batches
	var allEmbeddings [][]float32

	for i := 0; i < len(texts); i += e.maxBatchSize {
		end := i + e.maxBatchSize
		if end > len(texts) {
			end = len(texts)
		}

		subBatch := texts[i:end]

		// Double-check token count for this sub-batch
		estimatedTokens := e.estimateTokens(subBatch)
		if estimatedTokens > e.maxTokensHint {
			// If still too large, reduce batch size for this iteration
			reducedSize := (e.maxBatchSize * e.maxTokensHint) / estimatedTokens
			if reducedSize < 1 {
				reducedSize = 1
			}
			end = i + reducedSize
			if end > len(texts) {
				end = len(texts)
			}
			subBatch = texts[i:end]
		}

		embeddings, err := e.embedBatchDirect(ctx, subBatch)
		if err != nil {
			return nil, fmt.Errorf("failed to embed sub-batch [%d:%d]: %w", i, end, err)
		}

		allEmbeddings = append(allEmbeddings, embeddings...)
	}

	return allEmbeddings, nil
}

// estimateTokens provides a rough estimate of token count
// Rule of thumb: ~4 characters per token for English text
func (e *LocalEmbedder) estimateTokens(texts []string) int {
	totalChars := 0
	for _, text := range texts {
		totalChars += len(text)
	}
	// Add overhead for JSON structure (~100 chars per text)
	totalChars += len(texts) * 100
	return totalChars / 4
}

// embedBatchDirect sends a single batch request to the API
func (e *LocalEmbedder) embedBatchDirect(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody := EmbeddingRequest{
		Model: e.model,
		Input: texts,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LM Studio returned status %d: %s", resp.StatusCode, string(body))
	}

	var embResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	embeddings := make([][]float32, len(embResp.Data))
	for i, data := range embResp.Data {
		embeddings[i] = data.Embedding
	}

	return embeddings, nil
}

// Factory function qui choisit le bon embedder
func NewEmbedder(embedType, model, apiKey, baseURL string, dim int) (Embedder, error) {
	switch embedType {
	case "local", "lmstudio":
		return NewLocalEmbedder(baseURL, model, dim)
	case "openai":
		return NewOpenAIEmbedder(model, apiKey, dim)
	default:
		return nil, fmt.Errorf("unknown embedding type: %s", embedType)
	}
}
