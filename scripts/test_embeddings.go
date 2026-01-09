package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/Mirrdhyn/code-rag-mcp/config"
	"github.com/Mirrdhyn/code-rag-mcp/rag"
)

func main() {
	fmt.Println("🧪 Testing Embeddings System\n")
	fmt.Println("================================\n")

	// Load config
	cfg, err := config.Load("")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Configuration:\n")
	fmt.Printf("  Type: %s\n", cfg.EmbeddingType)
	fmt.Printf("  Model: %s\n", cfg.EmbeddingModel)
	fmt.Printf("  Dimension: %d\n", cfg.EmbeddingDim)
	fmt.Printf("  Base URL: %s\n\n", cfg.EmbeddingBaseURL)

	// Create embedder
	embedder, err := rag.NewEmbedder(
		cfg.EmbeddingType,
		cfg.EmbeddingModel,
		cfg.EmbeddingAPIKey,
		cfg.EmbeddingBaseURL,
		cfg.EmbeddingDim,
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Test 1: Single embedding
	fmt.Println("Test 1: Single Embedding")
	fmt.Println("-------------------------")
	testText := "func main() { fmt.Println(\"Hello World\") }"
	start := time.Now()
	vec, err := embedder.Embed(ctx, testText)
	if err != nil {
		log.Fatal(err)
	}
	duration := time.Since(start)

	fmt.Printf("✅ Success\n")
	fmt.Printf("   Text: %s\n", testText)
	fmt.Printf("   Dimensions: %d\n", len(vec))
	fmt.Printf("   Time: %v\n", duration)
	fmt.Printf("   First 5 values: [%.4f, %.4f, %.4f, %.4f, %.4f]\n\n",
		vec[0], vec[1], vec[2], vec[3], vec[4])

	// Test 2: Batch embeddings
	fmt.Println("Test 2: Batch Embeddings")
	fmt.Println("------------------------")
	texts := []string{
		"import tensorflow as tf",
		"package main",
		"const express = require('express')",
		"resource \"aws_vpc\" \"main\" {}",
		"def calculate_sum(a, b): return a + b",
	}

	start = time.Now()
	vecs, err := embedder.EmbedBatch(ctx, texts)
	if err != nil {
		log.Fatal(err)
	}
	duration = time.Since(start)

	fmt.Printf("✅ Success\n")
	fmt.Printf("   Batch size: %d\n", len(vecs))
	fmt.Printf("   Total time: %v\n", duration)
	fmt.Printf("   Avg time per embedding: %v\n\n", duration/time.Duration(len(texts)))

	// Test 3: Semantic similarity
	fmt.Println("Test 3: Semantic Similarity")
	fmt.Println("---------------------------")
	query := "golang http server"
	queryVec, _ := embedder.Embed(ctx, query)

	docs := []string{
		"func ServeHTTP(w http.ResponseWriter, r *http.Request)",
		"print('hello world')",
		"router.get('/api', handler)",
		"http.HandleFunc(\"/\", myHandler)",
		"SELECT * FROM users",
	}
	docVecs, _ := embedder.EmbedBatch(ctx, docs)

	fmt.Printf("Query: '%s'\n\n", query)
	fmt.Println("Similarity scores:")

	scores := make([]struct {
		text  string
		score float32
	}, len(docs))

	for i, doc := range docs {
		sim := cosineSimilarity(queryVec, docVecs[i])
		scores[i] = struct {
			text  string
			score float32
		}{doc, sim}
	}

	// Sort by score (descending)
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	for i, s := range scores {
		bar := ""
		for j := 0; j < int(s.score*50); j++ {
			bar += "█"
		}
		fmt.Printf("  %d. [%.3f] %s %s\n", i+1, s.score, bar, s.text)
	}

	fmt.Println("\n================================")
	fmt.Println("✅ All tests passed!")
}

func cosineSimilarity(a, b []float32) float32 {
	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	return dot / (sqrt(normA) * sqrt(normB))
}

func sqrt(x float32) float32 {
	return float32(math.Sqrt(float64(x)))
}
