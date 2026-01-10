package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
)

// HTTPAPIServer provides an HTTP API for triggering re-indexing from git hooks
type HTTPAPIServer struct {
	server  *RAGServer
	httpSrv *http.Server
	logger  *zap.Logger
	port    int
}

// ReindexRequest is the request body for the /reindex endpoint
type ReindexRequest struct {
	Files []string `json:"files"`
}

// ReindexResponse is the response body for the /reindex endpoint
type ReindexResponse struct {
	Success      bool     `json:"success"`
	Message      string   `json:"message"`
	FilesIndexed int      `json:"files_indexed"`
	Errors       []string `json:"errors,omitempty"`
}

// HealthResponse is the response body for the /health endpoint
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// NewHTTPAPIServer creates a new HTTP API server
func NewHTTPAPIServer(ragServer *RAGServer, port int, logger *zap.Logger) *HTTPAPIServer {
	return &HTTPAPIServer{
		server: ragServer,
		logger: logger,
		port:   port,
	}
}

// Start starts the HTTP API server in a goroutine
func (h *HTTPAPIServer) Start() error {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", h.handleHealth)

	// Reindex endpoint - accepts POST with file paths
	mux.HandleFunc("/reindex", h.handleReindex)

	// Reindex from marker file endpoint - reads .code-rag-pending-reindex
	mux.HandleFunc("/reindex-pending", h.handleReindexPending)

	h.httpSrv = &http.Server{
		Addr:         fmt.Sprintf(":%d", h.port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 300 * time.Second, // Long timeout for reindexing
	}

	go func() {
		h.logger.Info("HTTP API server starting", zap.Int("port", h.port))
		if err := h.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			h.logger.Error("HTTP API server error", zap.Error(err))
		}
	}()

	return nil
}

// Stop gracefully stops the HTTP API server
func (h *HTTPAPIServer) Stop(ctx context.Context) error {
	if h.httpSrv != nil {
		return h.httpSrv.Shutdown(ctx)
	}
	return nil
}

// handleHealth handles GET /health
func (h *HTTPAPIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := HealthResponse{
		Status:  "ok",
		Version: h.server.config.ServerVersion,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleReindex handles POST /reindex with JSON body containing file paths
func (h *HTTPAPIServer) handleReindex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ReindexRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode reindex request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Files) == 0 {
		http.Error(w, "No files specified", http.StatusBadRequest)
		return
	}

	h.logger.Info("Received reindex request", zap.Int("file_count", len(req.Files)))

	// Perform reindexing
	ctx := context.Background()
	var errors []string
	successCount := 0

	for _, filePath := range req.Files {
		filePath = strings.TrimSpace(filePath)
		if filePath == "" {
			continue
		}

		if err := h.server.incrementalIndexer.ReindexFiles(ctx, []string{filePath}, h.server.config.CollectionName); err != nil {
			h.logger.Error("Failed to reindex file", zap.String("file", filePath), zap.Error(err))
			errors = append(errors, fmt.Sprintf("%s: %v", filePath, err))
		} else {
			successCount++
		}
	}

	resp := ReindexResponse{
		Success:      len(errors) == 0,
		Message:      fmt.Sprintf("Reindexed %d/%d files", successCount, len(req.Files)),
		FilesIndexed: successCount,
		Errors:       errors,
	}

	w.Header().Set("Content-Type", "application/json")
	if len(errors) > 0 {
		w.WriteHeader(http.StatusPartialContent)
	}
	json.NewEncoder(w).Encode(resp)
}

// handleReindexPending handles POST /reindex-pending
// Reads the .code-rag-pending-reindex marker file and reindexes those files
func (h *HTTPAPIServer) handleReindexPending(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get working directory from query param or use current
	workDir := r.URL.Query().Get("workdir")
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			http.Error(w, "Failed to get working directory", http.StatusInternalServerError)
			return
		}
	}

	markerFile := workDir + "/.code-rag-pending-reindex"

	// Read marker file
	data, err := os.ReadFile(markerFile)
	if err != nil {
		if os.IsNotExist(err) {
			resp := ReindexResponse{
				Success:      true,
				Message:      "No pending reindex requests",
				FilesIndexed: 0,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		h.logger.Error("Failed to read marker file", zap.Error(err))
		http.Error(w, "Failed to read marker file", http.StatusInternalServerError)
		return
	}

	// Parse file paths
	var filePaths []string
	for _, path := range strings.Fields(string(data)) {
		path = strings.TrimSpace(path)
		if path != "" {
			filePaths = append(filePaths, path)
		}
	}

	if len(filePaths) == 0 {
		os.Remove(markerFile)
		resp := ReindexResponse{
			Success:      true,
			Message:      "Marker file was empty",
			FilesIndexed: 0,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	h.logger.Info("Processing pending reindex from marker file", zap.Int("file_count", len(filePaths)))

	// Perform reindexing
	ctx := context.Background()
	var errors []string
	successCount := 0

	for _, filePath := range filePaths {
		if err := h.server.incrementalIndexer.ReindexFiles(ctx, []string{filePath}, h.server.config.CollectionName); err != nil {
			h.logger.Error("Failed to reindex file", zap.String("file", filePath), zap.Error(err))
			errors = append(errors, fmt.Sprintf("%s: %v", filePath, err))
		} else {
			successCount++
		}
	}

	// Remove marker file on success
	if len(errors) == 0 {
		if err := os.Remove(markerFile); err != nil {
			h.logger.Warn("Failed to remove marker file", zap.Error(err))
		}
	}

	resp := ReindexResponse{
		Success:      len(errors) == 0,
		Message:      fmt.Sprintf("Reindexed %d/%d files", successCount, len(filePaths)),
		FilesIndexed: successCount,
		Errors:       errors,
	}

	w.Header().Set("Content-Type", "application/json")
	if len(errors) > 0 {
		w.WriteHeader(http.StatusPartialContent)
	}
	json.NewEncoder(w).Encode(resp)
}
