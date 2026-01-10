package main

import (
	"context"
	"flag"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Mirrdhyn/code-rag-mcp/config"
	"github.com/Mirrdhyn/code-rag-mcp/rag"
	"github.com/Mirrdhyn/code-rag-mcp/server"
	"go.uber.org/zap"
)

// processPendingReindex checks for pending re-index requests from git hooks
func processPendingReindex(workDir string, incrementalIndexer *rag.IncrementalIndexer, collectionName string, logger *zap.Logger) {
	markerFile := workDir + "/.code-rag-pending-reindex"

	// Check if marker file exists
	data, err := os.ReadFile(markerFile)
	if err != nil {
		// File doesn't exist or can't be read - no pending reindex
		return
	}

	logger.Info("Found pending re-index request from git hook")

	// Parse file paths from marker file (space-separated)
	filePaths := []string{}
	for _, path := range strings.Fields(string(data)) {
		path = strings.TrimSpace(path)
		if path != "" {
			filePaths = append(filePaths, path)
		}
	}

	if len(filePaths) == 0 {
		logger.Warn("Pending re-index marker file is empty")
		os.Remove(markerFile)
		return
	}

	logger.Info("Re-indexing files from git hook", zap.Int("file_count", len(filePaths)))

	// Re-index each file
	ctx := context.Background()
	successCount := 0
	for _, filePath := range filePaths {
		if err := incrementalIndexer.ReindexFiles(ctx, []string{filePath}, collectionName); err != nil {
			logger.Error("Failed to re-index file", zap.String("file", filePath), zap.Error(err))
		} else {
			successCount++
		}
	}

	logger.Info("Completed pending re-index",
		zap.Int("success", successCount),
		zap.Int("total", len(filePaths)))

	// Remove marker file
	if err := os.Remove(markerFile); err != nil {
		logger.Warn("Failed to remove marker file", zap.Error(err))
	}
}

func main() {
	configPath := flag.String("config", "", "Path to config file")
	flag.Parse()

	// Initialize logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	logger.Info("Starting Code RAG MCP Server",
		zap.String("embedding_type", cfg.EmbeddingType),
		zap.String("embedding_model", cfg.EmbeddingModel),
		zap.Int("embedding_dim", cfg.EmbeddingDim),
	)

	// Initialize embedder based on type
	embedder, err := rag.NewEmbedder(
		cfg.EmbeddingType,
		cfg.EmbeddingModel,
		cfg.EmbeddingAPIKey,
		cfg.EmbeddingBaseURL,
		cfg.EmbeddingDim,
	)
	if err != nil {
		logger.Fatal("Failed to create embedder", zap.Error(err))
	}

	logger.Info("Embedder initialized successfully", zap.Int("dimension", embedder.Dimension()))

	// Initialize vector database
	// Parse Qdrant URL to extract host and port
	host, portStr, err := net.SplitHostPort(cfg.QdrantURL)
	if err != nil {
		logger.Fatal("Failed to parse Qdrant URL", zap.Error(err))
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		logger.Fatal("Failed to parse Qdrant port", zap.Error(err))
	}

	vectorDB, err := rag.NewQdrantDB(host, port, cfg.QdrantAPIKey)
	if err != nil {
		logger.Fatal("Failed to connect to Qdrant", zap.Error(err))
	}
	defer vectorDB.Close()

	// Ensure collection exists with correct dimension
	ctx := context.Background()
	if err := vectorDB.CreateCollection(ctx, cfg.CollectionName, embedder.Dimension()); err != nil {
		logger.Warn("Collection might already exist", zap.Error(err))
	}

	// Initialize indexer
	indexer := rag.NewIndexer(embedder, vectorDB, logger)

	// Initialize incremental indexer
	workDir, _ := os.Getwd()
	incrementalIndexer := rag.NewIncrementalIndexer(indexer, workDir)

	// Process pending re-index requests from git hooks
	processPendingReindex(workDir, incrementalIndexer, cfg.CollectionName, logger)

	// Auto-index configured paths in background (if enabled)
	go func() {
		if cfg.AutoIndexOnStartup && len(cfg.CodePaths) > 0 {
			logger.Info("Starting background indexing", zap.Strings("paths", cfg.CodePaths))

			for _, path := range cfg.CodePaths {
				if _, err := os.Stat(path); os.IsNotExist(err) {
					logger.Warn("Skipping non-existent path", zap.String("path", path))
					continue
				}

				logger.Info("Indexing path", zap.String("path", path))
				if err := incrementalIndexer.IndexDirectoryIncremental(
					context.Background(),
					path,
					cfg.FileExtensions,
					cfg.CollectionName,
				); err != nil {
					logger.Error("Background indexing failed", zap.String("path", path), zap.Error(err))
				}
			}

			logger.Info("Background indexing complete")
		}
	}()

	// Create MCP server
	mcpServer := server.NewRAGServer(indexer, incrementalIndexer, vectorDB, embedder, cfg, logger)

	// Start HTTP API server if enabled
	var httpAPIServer *server.HTTPAPIServer
	if cfg.HTTPAPIEnabled {
		httpAPIServer = server.NewHTTPAPIServer(mcpServer, cfg.HTTPAPIPort, logger)
		if err := httpAPIServer.Start(); err != nil {
			logger.Error("Failed to start HTTP API server", zap.Error(err))
		} else {
			logger.Info("HTTP API server started",
				zap.Int("port", cfg.HTTPAPIPort),
				zap.String("reindex_endpoint", "POST /reindex"),
				zap.String("reindex_pending_endpoint", "POST /reindex-pending"),
			)
		}
	}

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Shutting down gracefully...")
		
		// Stop HTTP API server
		if httpAPIServer != nil {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			if err := httpAPIServer.Stop(shutdownCtx); err != nil {
				logger.Error("Failed to stop HTTP API server", zap.Error(err))
			}
		}
		
		cancel()
	}()

	logger.Info("MCP Server starting...")
	if err := mcpServer.Serve(ctx); err != nil {
		logger.Fatal("Server error", zap.Error(err))
	}
}
