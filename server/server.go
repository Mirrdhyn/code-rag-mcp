package server

import (
	"context"

	"github.com/Mirrdhyn/code-rag-mcp/config"
	"github.com/Mirrdhyn/code-rag-mcp/rag"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
)

type RAGServer struct {
	mcp                *server.MCPServer
	indexer            *rag.Indexer
	incrementalIndexer *rag.IncrementalIndexer
	vectorDB           rag.VectorDB
	embedder           rag.Embedder
	config             *config.Config
	logger             *zap.Logger
}

func NewRAGServer(indexer *rag.Indexer, incrementalIndexer *rag.IncrementalIndexer, vectorDB rag.VectorDB, embedder rag.Embedder, cfg *config.Config, logger *zap.Logger) *RAGServer {
	s := &RAGServer{
		indexer:            indexer,
		incrementalIndexer: incrementalIndexer,
		vectorDB:           vectorDB,
		embedder:           embedder,
		config:             cfg,
		logger:             logger,
	}

	mcpServer := server.NewMCPServer(
		cfg.ServerName,
		cfg.ServerVersion,
	)

	s.registerTools(mcpServer)
	s.mcp = mcpServer

	return s
}

func (s *RAGServer) Serve(ctx context.Context) error {
	return server.ServeStdio(s.mcp)
}
