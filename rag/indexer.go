package rag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Indexer struct {
	embedder Embedder
	vectorDB VectorDB
	logger   *zap.Logger
}

type CodeChunk struct {
	FilePath  string
	Content   string
	LineStart int
	LineEnd   int
	Language  string
}

func NewIndexer(embedder Embedder, vectorDB VectorDB, logger *zap.Logger) *Indexer {
	return &Indexer{
		embedder: embedder,
		vectorDB: vectorDB,
		logger:   logger,
	}
}

func (idx *Indexer) IndexDirectory(ctx context.Context, path string, extensions []string, collectionName string) error {
	idx.logger.Info("Starting indexing", zap.String("path", path))

	var chunks []CodeChunk

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Skip certain directories
			dirName := filepath.Base(filePath)
			if dirName == "node_modules" ||
				dirName == ".git" ||
				dirName == "vendor" ||
				dirName == "__pycache__" ||
				dirName == ".venv" ||
				dirName == "venv" ||
				strings.HasPrefix(dirName, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Check extension
		ext := filepath.Ext(filePath)
		if !contains(extensions, ext) {
			return nil
		}

		// Skip large files
		if info.Size() > 1024*1024 { // 1MB
			idx.logger.Debug("Skipping large file", zap.String("file", filePath), zap.Int64("size", info.Size()))
			return nil
		}

		// Read and chunk file
		fileChunks, err := idx.chunkFile(filePath)
		if err != nil {
			idx.logger.Warn("Failed to chunk file", zap.String("file", filePath), zap.Error(err))
			return nil
		}

		chunks = append(chunks, fileChunks...)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	idx.logger.Info("Chunked files", zap.Int("chunks", len(chunks)))

	if len(chunks) == 0 {
		return fmt.Errorf("no files found to index")
	}

	// Embed chunks in batches
	batchSize := 100
	for i := 0; i < len(chunks); i += batchSize {
		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}

		batch := chunks[i:end]
		if err := idx.indexBatch(ctx, batch, collectionName); err != nil {
			return fmt.Errorf("failed to index batch: %w", err)
		}

		idx.logger.Info("Indexed batch", zap.Int("start", i), zap.Int("end", end), zap.Int("total", len(chunks)))
	}

	idx.logger.Info("Indexing complete", zap.Int("total_chunks", len(chunks)))
	return nil
}

func (idx *Indexer) chunkFile(filePath string) ([]CodeChunk, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	text := string(content)
	lines := strings.Split(text, "\n")

	// Simple chunking by line count
	chunkSize := 50
	overlap := 10

	var chunks []CodeChunk
	for i := 0; i < len(lines); i += chunkSize - overlap {
		end := i + chunkSize
		if end > len(lines) {
			end = len(lines)
		}

		chunkText := strings.Join(lines[i:end], "\n")
		if strings.TrimSpace(chunkText) == "" {
			continue
		}

		chunks = append(chunks, CodeChunk{
			FilePath:  filePath,
			Content:   chunkText,
			LineStart: i + 1,
			LineEnd:   end,
			Language:  detectLanguage(filePath),
		})

		if end == len(lines) {
			break
		}
	}

	return chunks, nil
}

func (idx *Indexer) indexBatch(ctx context.Context, chunks []CodeChunk, collectionName string) error {
	// Extract texts for embedding
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		// Enhance text with context for better embeddings
		texts[i] = fmt.Sprintf("File: %s\nLanguage: %s\nCode:\n%s",
			filepath.Base(chunk.FilePath),
			chunk.Language,
			chunk.Content,
		)
	}

	// Generate embeddings
	embeddings, err := idx.embedder.EmbedBatch(ctx, texts)
	if err != nil {
		return err
	}

	// Create points
	points := make([]Point, len(chunks))
	for i, chunk := range chunks {
		points[i] = Point{
			ID:     uuid.New().String(),
			Vector: embeddings[i],
			Payload: map[string]interface{}{
				"file_path":  chunk.FilePath,
				"content":    chunk.Content,
				"line_start": chunk.LineStart,
				"line_end":   chunk.LineEnd,
				"language":   chunk.Language,
			},
		}
	}

	// Upsert to vector DB
	return idx.vectorDB.Upsert(ctx, collectionName, points)
}

func detectLanguage(filePath string) string {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".tf":
		return "terraform"
	case ".yaml", ".yml":
		return "yaml"
	case ".md":
		return "markdown"
	case ".json":
		return "json"
	case ".sh":
		return "bash"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".c", ".h":
		return "c"
	case ".cpp", ".hpp", ".cc":
		return "cpp"
	default:
		return "unknown"
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ReindexFiles re-indexes specific files (used by git hooks)
func (idx *Indexer) ReindexFiles(ctx context.Context, filePaths []string, collectionName string) error {
	idx.logger.Info("Re-indexing files", zap.Int("count", len(filePaths)), zap.Strings("files", filePaths))

	var allChunks []CodeChunk
	deletedCount := 0
	indexedCount := 0

	for _, filePath := range filePaths {
		// Delete old chunks for this file
		err := idx.vectorDB.Delete(ctx, collectionName, map[string]interface{}{
			"file_path": filePath,
		})
		if err != nil {
			idx.logger.Warn("Failed to delete old chunks", zap.String("file", filePath), zap.Error(err))
		} else {
			deletedCount++
		}

		// Check if file still exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			idx.logger.Info("File deleted, skipping re-indexing", zap.String("file", filePath))
			continue
		}

		// Re-chunk and prepare for indexing
		chunks, err := idx.chunkFile(filePath)
		if err != nil {
			idx.logger.Warn("Failed to chunk file", zap.String("file", filePath), zap.Error(err))
			continue
		}

		if len(chunks) == 0 {
			idx.logger.Debug("No chunks generated", zap.String("file", filePath))
			continue
		}

		allChunks = append(allChunks, chunks...)
		indexedCount++
	}

	if len(allChunks) == 0 {
		idx.logger.Info("No chunks to re-index")
		return nil
	}

	// Index all new chunks
	if err := idx.indexBatch(ctx, allChunks, collectionName); err != nil {
		return fmt.Errorf("failed to index chunks: %w", err)
	}

	idx.logger.Info("Re-indexing complete",
		zap.Int("files_processed", indexedCount),
		zap.Int("files_deleted", deletedCount),
		zap.Int("total_chunks", len(allChunks)),
	)

	return nil
}
