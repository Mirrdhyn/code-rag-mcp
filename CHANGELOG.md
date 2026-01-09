# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-01-09

### Added
- Initial release of Code RAG MCP Server
- Semantic code search using vector embeddings
- Support for local embeddings via LM Studio
- Support for OpenAI embeddings
- Qdrant vector database integration
- MCP (Model Context Protocol) server implementation
- Multi-language support: Go, Python, JavaScript/TypeScript, Terraform, YAML, Markdown, JSON, Shell
- Incremental indexing with state tracking
- Git hooks for automatic re-indexing (post-commit, post-merge)
- MCP tools:
  - `semantic_code_search` - Find code by concept
  - `find_similar_code` - Find similar code snippets
  - `explain_code_with_context` - Explain code with context
  - `index_codebase` - Index directories
  - `reindex_files` - Re-index specific files
  - `get_index_stats` - Check index statistics
  - `get_indexing_progress` - Monitor background indexing
- Integration examples for Claude Desktop, Claude Code, and Zed
- Comprehensive documentation and quick start guide
- Configurable chunking and search parameters
- Automatic collection creation in Qdrant

### Features
- Fast semantic search replacing traditional grep/ripgrep
- Context-aware code understanding
- Efficient vector similarity search
- Background indexing on startup
- Graceful shutdown handling
- Extensive logging with zap

[1.0.0]: https://github.com/YOUR_USERNAME/code-rag-mcp/releases/tag/v1.0.0
