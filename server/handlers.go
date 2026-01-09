package server

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
)

func (s *RAGServer) handleSemanticSearch(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	query, ok := arguments["query"].(string)
	if !ok {
		return mcp.NewToolResultError("query must be a string"), nil
	}

	limit := 5
	if l, ok := arguments["limit"].(float64); ok {
		limit = int(l)
	}

	minScore := float32(0.15) // Lowered for high-dim embeddings (3584)
	if ms, ok := arguments["min_score"].(float64); ok {
		minScore = float32(ms)
	}

	compact := true // Default to compact mode to save tokens
	if c, ok := arguments["compact"].(bool); ok {
		compact = c
	}

	excerptLines := 0 // 0 = full chunk
	if el, ok := arguments["excerpt_lines"].(float64); ok {
		excerptLines = int(el)
	}

	ctx := context.Background()

	s.logger.Info("Semantic search",
		zap.String("query", query),
		zap.Int("limit", limit),
		zap.Float32("min_score", minScore),
		zap.Bool("compact", compact),
		zap.Int("excerpt_lines", excerptLines),
	)

	// Generate embedding for query
	embedding, err := s.embedder.Embed(ctx, query)
	if err != nil {
		s.logger.Error("Failed to generate embedding", zap.Error(err))
		return mcp.NewToolResultError(fmt.Sprintf("Failed to generate embedding: %v", err)), nil
	}

	// Search vector DB
	results, err := s.vectorDB.Search(ctx, s.config.CollectionName, embedding, limit, minScore)
	if err != nil {
		s.logger.Error("Search failed", zap.Error(err))
		return mcp.NewToolResultError(fmt.Sprintf("Search failed: %v", err)), nil
	}

	if len(results) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No results found for query: '%s'\n\nTry:\n- Lowering min_score to 0.5-0.6\n- Broader query terms\n- Check if codebase is indexed", query)), nil
	}

	// Format results based on mode
	var output strings.Builder
	output.WriteString(fmt.Sprintf("# Semantic Search Results\n\n"))
	output.WriteString(fmt.Sprintf("Query: **%s**\n", query))
	output.WriteString(fmt.Sprintf("Found: **%d matches** (deduplicated)\n\n", len(results)))

	if compact {
		output.WriteString("💡 **Compact mode** - showing file:line references only\n\n")
		output.WriteString("---\n\n")

		for i, result := range results {
			output.WriteString(fmt.Sprintf("%d. `%s:%d-%d` (Score: %.3f, %s)\n",
				i+1, result.FilePath, result.LineStart, result.LineEnd, result.Score, result.Language))
		}

		output.WriteString("\n💡 Use `compact: false` to see full code excerpts.\n")
	} else {
		output.WriteString("---\n\n")

		for i, result := range results {
			output.WriteString(fmt.Sprintf("## %d. %s (Score: %.3f)\n\n", i+1, result.FilePath, result.Score))
			output.WriteString(fmt.Sprintf("**Language:** %s | **Lines:** %d-%d\n\n", result.Language, result.LineStart, result.LineEnd))

			// Truncate content if excerpt_lines is set
			content := result.Content
			if excerptLines > 0 {
				lines := strings.Split(content, "\n")
				if len(lines) > excerptLines {
					content = strings.Join(lines[:excerptLines], "\n")
					content += fmt.Sprintf("\n... (%d more lines)", len(lines)-excerptLines)
				}
			}

			output.WriteString("```" + result.Language + "\n")
			output.WriteString(content)
			output.WriteString("\n```\n\n")
		}

		if excerptLines == 0 {
			output.WriteString("💡 **Tip:** Use `excerpt_lines: 15` to show only first 15 lines and save tokens.\n")
		}
	}

	return mcp.NewToolResultText(output.String()), nil
}

func (s *RAGServer) handleFindSimilarCode(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	snippet, ok := arguments["code_snippet"].(string)
	if !ok {
		return mcp.NewToolResultError("code_snippet must be a string"), nil
	}

	limit := 5
	if l, ok := arguments["limit"].(float64); ok {
		limit = int(l)
	}

	minScore := float32(0.18) // Lowered for high-dim embeddings (3584)
	if ms, ok := arguments["min_score"].(float64); ok {
		minScore = float32(ms)
	}

	ctx := context.Background()

	s.logger.Info("Finding similar code", zap.Int("snippet_length", len(snippet)), zap.Int("limit", limit))

	// Generate embedding for snippet
	embedding, err := s.embedder.Embed(ctx, snippet)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to generate embedding: %v", err)), nil
	}

	// Search
	results, err := s.vectorDB.Search(ctx, s.config.CollectionName, embedding, limit, minScore)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Search failed: %v", err)), nil
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("# Similar Code Matches\n\n"))
	output.WriteString(fmt.Sprintf("Found: **%d similar snippets**\n\n", len(results)))
	output.WriteString("---\n\n")

	for i, result := range results {
		output.WriteString(fmt.Sprintf("## Match %d (Similarity: %.1f%%)\n\n", i+1, result.Score*100))
		output.WriteString(fmt.Sprintf("**File:** %s | **Lines:** %d-%d\n\n", result.FilePath, result.LineStart, result.LineEnd))
		output.WriteString("```" + result.Language + "\n")
		output.WriteString(result.Content)
		output.WriteString("\n```\n\n")
	}

	return mcp.NewToolResultText(output.String()), nil
}

func (s *RAGServer) handleExplainCode(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	filePath, ok := arguments["file_path"].(string)
	if !ok {
		return mcp.NewToolResultError("file_path must be a string"), nil
	}

	focus := ""
	if f, ok := arguments["focus"].(string); ok {
		focus = f
	}

	ctx := context.Background()

	s.logger.Info("Explaining code", zap.String("file", filePath), zap.String("focus", focus))

	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to read file: %v", err)), nil
	}

	// Search for related code
	query := fmt.Sprintf("code related to %s %s", filePath, focus)
	embedding, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to generate embedding: %v", err)), nil
	}

	results, err := s.vectorDB.Search(ctx, s.config.CollectionName, embedding, 5, 0.6)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Search failed: %v", err)), nil
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("# Code Explanation: %s\n\n", filePath))

	output.WriteString("## Main Code\n\n")
	output.WriteString("```\n")
	output.WriteString(string(content))
	output.WriteString("\n```\n\n")

	if len(results) > 0 {
		output.WriteString("## Related Context\n\n")
		for i, result := range results {
			if result.FilePath == filePath {
				continue // Skip same file
			}
			output.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, result.FilePath))
			output.WriteString("```" + result.Language + "\n")
			output.WriteString(result.Content)
			output.WriteString("\n```\n\n")
		}
	}

	return mcp.NewToolResultText(output.String()), nil
}

func (s *RAGServer) handleIndexDirectory(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	path, ok := arguments["path"].(string)
	if !ok {
		return mcp.NewToolResultError("path must be a string"), nil
	}

	extensions := s.config.FileExtensions
	if exts, ok := arguments["extensions"].([]interface{}); ok {
		extensions = make([]string, len(exts))
		for i, ext := range exts {
			extensions[i] = ext.(string)
		}
	}

	ctx := context.Background()

	s.logger.Info("Starting indexing", zap.String("path", path), zap.Strings("extensions", extensions))

	// Check if path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return mcp.NewToolResultError(fmt.Sprintf("Path does not exist: %s", path)), nil
	}

	err := s.indexer.IndexDirectory(ctx, path, extensions, s.config.CollectionName)
	if err != nil {
		s.logger.Error("Indexing failed", zap.Error(err))
		return mcp.NewToolResultError(fmt.Sprintf("Indexing failed: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("✅ Successfully indexed directory: %s\n\nThe codebase is now ready for semantic search!", path)), nil
}

func (s *RAGServer) handleGetStats(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	ctx := context.Background()

	// Get collection info from Qdrant
	info, err := s.vectorDB.GetCollectionInfo(ctx, s.config.CollectionName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get stats: %v", err)), nil
	}

	output := fmt.Sprintf(`# Semantic Search Index Statistics

**Status:** ✅ Ready
**Total Code Chunks:** %d
**Vector Dimension:** %d
**Embedding Model:** %s (%s)
**Last Updated:** %s

**Indexed Languages:**
- Go, Python, JavaScript/TypeScript
- Terraform, YAML, Markdown
- And more...

**Configuration:**
- Chunk Size: %d lines
- Chunk Overlap: %d lines
- Min Score: %.2f

💡 **The index is ready!** Use 'semantic_code_search' to find code by concept.

**Example queries:**
- "authentication middleware"
- "database connection logic"
- "error handling patterns"
- "terraform AWS VPC configuration"
`,
		info.PointsCount,
		info.VectorDim,
		s.config.EmbeddingModel,
		s.config.EmbeddingType,
		info.UpdatedAt.Format("2006-01-02 15:04:05"),
		s.config.ChunkSize,
		s.config.ChunkOverlap,
		s.config.MinScore,
	)

	return mcp.NewToolResultText(output), nil
}

func (s *RAGServer) handleReindexFiles(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	filePathsRaw, ok := arguments["file_paths"].([]interface{})
	if !ok {
		return mcp.NewToolResultError("file_paths must be an array of strings"), nil
	}

	filePaths := make([]string, len(filePathsRaw))
	for i, fp := range filePathsRaw {
		filePaths[i] = fp.(string)
	}

	if len(filePaths) == 0 {
		return mcp.NewToolResultError("file_paths cannot be empty"), nil
	}

	ctx := context.Background()

	s.logger.Info("Re-indexing files via MCP", zap.Strings("files", filePaths))

	err := s.indexer.ReindexFiles(ctx, filePaths, s.config.CollectionName)
	if err != nil {
		s.logger.Error("Re-indexing failed", zap.Error(err))
		return mcp.NewToolResultError(fmt.Sprintf("Re-indexing failed: %v", err)), nil
	}

	output := fmt.Sprintf(`✅ **Re-indexing complete!**

**Files processed:** %d

The index has been updated with the latest changes from these files.

**Files:**
`, len(filePaths))

	for _, fp := range filePaths {
		output += fmt.Sprintf("- %s\n", fp)
	}

	return mcp.NewToolResultText(output), nil
}

func (s *RAGServer) handleGetIndexingProgress(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	state := s.incrementalIndexer.GetState()

	if state == nil {
		return mcp.NewToolResultText("ℹ️ No active indexing session. Index is either complete or hasn't started yet."), nil
	}

	stats := state.GetStats()

	var statusEmoji string
	switch stats["status"].(string) {
	case "in_progress":
		statusEmoji = "⏳"
	case "completed":
		statusEmoji = "✅"
	case "failed":
		statusEmoji = "❌"
	default:
		statusEmoji = "❓"
	}

	output := fmt.Sprintf(`# Background Indexing Progress

**Status:** %s %s
**Root Path:** %s
**Progress:** %.1f%% (%d / %d files)
**Total Chunks Indexed:** %d
**Failed Files:** %d

**Timing:**
- Started: %s
- Last Update: %s
`,
		statusEmoji,
		stats["status"],
		stats["root_path"],
		stats["progress"],
		stats["indexed_files"],
		stats["total_files"],
		stats["total_chunks"],
		stats["failed_files"],
		stats["start_time"].(time.Time).Format("2006-01-02 15:04:05"),
		stats["last_update"].(time.Time).Format("2006-01-02 15:04:05"),
	)

	if duration, ok := stats["duration"].(string); ok {
		output += fmt.Sprintf("- Duration: %s\n", duration)
	}

	if stats["failed_files"].(int) > 0 {
		output += "\n⚠️ **Some files failed to index.** Check `.indexing_state.json` for details.\n"
	}

	if stats["status"].(string) == "in_progress" {
		output += "\n💡 **Tip:** You can use semantic search while indexing is in progress. Results will improve as more files are indexed.\n"
	} else if stats["status"].(string) == "completed" {
		output += "\n🎉 **Indexing complete!** Your codebase is fully searchable.\n"
	}

	return mcp.NewToolResultText(output), nil
}
