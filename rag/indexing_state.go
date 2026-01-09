package rag

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// IndexingState tracks the progress of indexing operations
type IndexingState struct {
	mu             sync.RWMutex
	RootPath       string            `json:"root_path"`
	TotalFiles     int               `json:"total_files"`
	IndexedFiles   int               `json:"indexed_files"`
	TotalChunks    int               `json:"total_chunks"`
	ProcessedFiles map[string]bool   `json:"processed_files"`
	FailedFiles    map[string]string `json:"failed_files"` // file -> error message
	LastUpdate     time.Time         `json:"last_update"`
	Status         string            `json:"status"` // "in_progress", "completed", "failed"
	StartTime      time.Time         `json:"start_time"`
	CompletionTime *time.Time        `json:"completion_time,omitempty"`
}

// NewIndexingState creates a new indexing state
func NewIndexingState(rootPath string) *IndexingState {
	return &IndexingState{
		RootPath:       rootPath,
		ProcessedFiles: make(map[string]bool),
		FailedFiles:    make(map[string]string),
		Status:         "in_progress",
		StartTime:      time.Now(),
		LastUpdate:     time.Now(),
	}
}

// LoadIndexingState loads the state from a JSON file
func LoadIndexingState(path string) (*IndexingState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var state IndexingState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// Save persists the state to a JSON file
func (s *IndexingState) Save(path string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// MarkFileProcessed marks a file as successfully processed
func (s *IndexingState) MarkFileProcessed(filePath string, chunksCount int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ProcessedFiles[filePath] = true
	s.IndexedFiles++
	s.TotalChunks += chunksCount
	s.LastUpdate = time.Now()
}

// MarkFileFailed marks a file as failed with an error message
func (s *IndexingState) MarkFileFailed(filePath string, errorMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.FailedFiles[filePath] = errorMsg
	s.LastUpdate = time.Now()
}

// IsFileProcessed checks if a file has already been processed
func (s *IndexingState) IsFileProcessed(filePath string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.ProcessedFiles[filePath]
}

// SetStatus updates the indexing status
func (s *IndexingState) SetStatus(status string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Status = status
	s.LastUpdate = time.Now()

	if status == "completed" || status == "failed" {
		now := time.Now()
		s.CompletionTime = &now
	}
}

// GetProgress returns the current progress percentage
func (s *IndexingState) GetProgress() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.TotalFiles == 0 {
		return 0.0
	}
	return float64(s.IndexedFiles) / float64(s.TotalFiles) * 100
}

// GetStats returns current statistics
func (s *IndexingState) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"root_path":     s.RootPath,
		"total_files":   s.TotalFiles,
		"indexed_files": s.IndexedFiles,
		"failed_files":  len(s.FailedFiles),
		"total_chunks":  s.TotalChunks,
		"progress":      s.GetProgress(),
		"status":        s.Status,
		"start_time":    s.StartTime,
		"last_update":   s.LastUpdate,
	}

	if s.CompletionTime != nil {
		stats["completion_time"] = s.CompletionTime
		stats["duration"] = s.CompletionTime.Sub(s.StartTime).String()
	}

	return stats
}
