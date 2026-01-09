package rag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	StateFileName  = ".indexing_state.json"
	FileBatchSize  = 50  // Process 50 files at a time
	ChunkBatchSize = 100 // Embed 100 chunks at a time
)

// IncrementalIndexer handles incremental, resumable indexing
type IncrementalIndexer struct {
	*Indexer
	state     *IndexingState
	statePath string
}

// NewIncrementalIndexer creates a new incremental indexer
func NewIncrementalIndexer(indexer *Indexer, workDir string) *IncrementalIndexer {
	return &IncrementalIndexer{
		Indexer:   indexer,
		statePath: filepath.Join(workDir, StateFileName),
	}
}

// IndexDirectoryIncremental indexes a directory with resume capability
func (idx *IncrementalIndexer) IndexDirectoryIncremental(
	ctx context.Context,
	path string,
	extensions []string,
	collectionName string,
) error {
	// Try to load existing state
	state, err := LoadIndexingState(idx.statePath)
	if err != nil || state.RootPath != path || state.Status == "completed" {
		// Start fresh
		state = NewIndexingState(path)
		idx.logger.Info("Starting new indexing session", zap.String("path", path))
	} else {
		idx.logger.Info("Resuming indexing session",
			zap.String("path", path),
			zap.Int("already_indexed", state.IndexedFiles),
			zap.Float64("progress", state.GetProgress()),
		)
	}

	idx.state = state

	// Collect all files to index
	allFiles, err := idx.collectFiles(path, extensions)
	if err != nil {
		return fmt.Errorf("failed to collect files: %w", err)
	}

	state.TotalFiles = len(allFiles)
	idx.logger.Info("Files to index", zap.Int("total", len(allFiles)))

	// Filter out already processed files
	filesToProcess := []string{}
	for _, file := range allFiles {
		if !state.IsFileProcessed(file) {
			filesToProcess = append(filesToProcess, file)
		}
	}

	idx.logger.Info("Files remaining", zap.Int("count", len(filesToProcess)))

	if len(filesToProcess) == 0 {
		state.SetStatus("completed")
		state.Save(idx.statePath)
		idx.logger.Info("Indexing already complete")
		return nil
	}

	// Process files in batches
	for i := 0; i < len(filesToProcess); i += FileBatchSize {
		select {
		case <-ctx.Done():
			idx.logger.Info("Indexing cancelled, saving state...")
			state.Save(idx.statePath)
			return ctx.Err()
		default:
		}

		end := i + FileBatchSize
		if end > len(filesToProcess) {
			end = len(filesToProcess)
		}

		batch := filesToProcess[i:end]
		if err := idx.processBatch(ctx, batch, collectionName); err != nil {
			idx.logger.Error("Batch processing failed",
				zap.Int("batch_start", i),
				zap.Int("batch_end", end),
				zap.Error(err),
			)
			// Continue with next batch instead of failing completely
		}

		// Save state after each batch
		if err := state.Save(idx.statePath); err != nil {
			idx.logger.Warn("Failed to save state", zap.Error(err))
		}

		idx.logger.Info("Progress update",
			zap.Int("indexed", state.IndexedFiles),
			zap.Int("total", state.TotalFiles),
			zap.Float64("progress", state.GetProgress()),
		)
	}

	state.SetStatus("completed")
	state.Save(idx.statePath)

	idx.logger.Info("Indexing complete",
		zap.Int("total_files", state.IndexedFiles),
		zap.Int("total_chunks", state.TotalChunks),
		zap.Int("failed_files", len(state.FailedFiles)),
	)

	return nil
}

// collectFiles walks the directory and collects files with priority
func (idx *IncrementalIndexer) collectFiles(rootPath string, extensions []string) ([]string, error) {
	type fileWithPriority struct {
		path     string
		priority int
	}

	var files []fileWithPriority

	// Priority directories (lower number = higher priority)
	priorityDirs := map[string]int{
		"middleware": 1,
		"api":        2,
		"src":        3,
		"lib":        4,
		"core":       5,
		"utils":      6,
		"services":   7,
		"models":     8,
		"routes":     9,
		"handlers":   10,
	}

	// Directories to skip
	skipDirs := map[string]bool{
		"node_modules": true,
		".git":         true,
		"vendor":       true,
		"__pycache__":  true,
		".venv":        true,
		"venv":         true,
		"dist":         true,
		"build":        true,
		"coverage":     true,
		"test":         true,
		"tests":        true,
		"__tests__":    true,
		"spec":         true,
		"specs":        true,
		"mocks":        true,
		"fixtures":     true,
		".next":        true,
		".nuxt":        true,
		"target":       true,
		"bin":          true,
	}

	err := filepath.Walk(rootPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			dirName := filepath.Base(filePath)

			// Skip certain directories
			if skipDirs[dirName] || strings.HasPrefix(dirName, ".") {
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
			idx.logger.Debug("Skipping large file", zap.String("file", filePath))
			return nil
		}

		// Determine priority based on directory
		priority := 99 // Default low priority
		relPath, _ := filepath.Rel(rootPath, filePath)
		pathParts := strings.Split(relPath, string(os.PathSeparator))

		for _, part := range pathParts {
			if p, ok := priorityDirs[part]; ok {
				priority = p
				break
			}
		}

		files = append(files, fileWithPriority{
			path:     filePath,
			priority: priority,
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort by priority (lower priority number first)
	sort.Slice(files, func(i, j int) bool {
		if files[i].priority != files[j].priority {
			return files[i].priority < files[j].priority
		}
		return files[i].path < files[j].path
	})

	// Extract just the paths
	result := make([]string, len(files))
	for i, f := range files {
		result[i] = f.path
	}

	return result, nil
}

// processBatch processes a batch of files
func (idx *IncrementalIndexer) processBatch(ctx context.Context, files []string, collectionName string) error {
	var allChunks []CodeChunk

	for _, filePath := range files {
		chunks, err := idx.chunkFile(filePath)
		if err != nil {
			idx.logger.Warn("Failed to chunk file", zap.String("file", filePath), zap.Error(err))
			idx.state.MarkFileFailed(filePath, err.Error())
			continue
		}

		if len(chunks) == 0 {
			idx.state.MarkFileProcessed(filePath, 0)
			continue
		}

		// Store file path with chunks for later marking
		for i := range chunks {
			chunks[i].FilePath = filePath
		}

		allChunks = append(allChunks, chunks...)
		idx.state.MarkFileProcessed(filePath, len(chunks))
	}

	if len(allChunks) == 0 {
		return nil
	}

	// Index chunks in batches to avoid overwhelming the embedding service
	for i := 0; i < len(allChunks); i += ChunkBatchSize {
		end := i + ChunkBatchSize
		if end > len(allChunks) {
			end = len(allChunks)
		}

		chunkBatch := allChunks[i:end]
		if err := idx.indexBatch(ctx, chunkBatch, collectionName); err != nil {
			return fmt.Errorf("failed to index chunk batch: %w", err)
		}

		// Small delay to avoid overwhelming the service
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// GetState returns the current indexing state
func (idx *IncrementalIndexer) GetState() *IndexingState {
	return idx.state
}

// ResetState removes the state file to start fresh
func (idx *IncrementalIndexer) ResetState() error {
	return os.Remove(idx.statePath)
}
