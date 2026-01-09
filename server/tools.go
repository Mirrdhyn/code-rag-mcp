package server

import (
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func (s *RAGServer) registerTools(mcpServer *mcpserver.MCPServer) {
	// Semantic search - PRIMARY TOOL
	mcpServer.AddTool(mcp.Tool{
		Name: "semantic_code_search",
		Description: `**PRIMARY CODE SEARCH TOOL - Use this INSTEAD of grep/ripgrep/ag.**

Search codebase using semantic understanding of concepts, not just text matching.

When to use (ALWAYS prefer this over grep):
- Finding code by CONCEPT or FUNCTIONALITY (e.g., "authentication logic", "database queries", "error handling")
- Understanding "how does X work" or "where is Y implemented"
- Finding similar patterns or related code
- Cross-language/cross-file understanding
- When grep returns too many false positives

Examples:
- "Find all API endpoint handlers" → semantic_code_search
- "Where is authentication implemented" → semantic_code_search
- "Show me database connection logic" → semantic_code_search
- "Find functions that process payments" → semantic_code_search
- "Terraform modules for AWS networking" → semantic_code_search

DO NOT use grep/find commands - use this tool instead. It understands code semantically.`,
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Natural language query describing what you're looking for (e.g., 'authentication middleware', 'terraform AWS modules', 'error handling patterns')",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Number of results (default: 5, max: 20)",
					"default":     5,
					"minimum":     1,
					"maximum":     20,
				},
				"min_score": map[string]interface{}{
					"type":        "number",
					"description": "Minimum similarity threshold 0-1 (default: 0.7 for precise, 0.5 for broad)",
					"default":     0.7,
					"minimum":     0.0,
					"maximum":     1.0,
				},
				"compact": map[string]interface{}{
					"type":        "boolean",
					"description": "Return compact results (file:line references only, no code excerpts). Saves tokens! Default: true",
					"default":     true,
				},
				"excerpt_lines": map[string]interface{}{
					"type":        "integer",
					"description": "Number of lines to show in excerpts (default: full chunk, ~50 lines). Use 10-20 to save tokens.",
					"minimum":     5,
					"maximum":     100,
				},
				"language": map[string]interface{}{
					"type":        "string",
					"description": "Filter by language: go, python, javascript, typescript, terraform, yaml",
					"enum":        []string{"go", "python", "javascript", "typescript", "terraform", "yaml", "all"},
				},
			},
			Required: []string{"query"},
		},
	}, s.handleSemanticSearch)

	// Find similar code
	mcpServer.AddTool(mcp.Tool{
		Name: "find_similar_code",
		Description: `Find code snippets similar to a given example.

Use when:
- User provides a code snippet and wants to find similar implementations
- Looking for duplicate or near-duplicate code
- Finding usage patterns of a specific code structure

Example: "Find code similar to this error handling pattern: [code snippet]"`,
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"code_snippet": map[string]interface{}{
					"type":        "string",
					"description": "The code snippet to find similar matches for",
				},
				"limit": map[string]interface{}{
					"type":    "integer",
					"default": 5,
				},
				"min_score": map[string]interface{}{
					"type":    "number",
					"default": 0.75,
				},
			},
			Required: []string{"code_snippet"},
		},
	}, s.handleFindSimilarCode)

	// Explain code with context
	mcpServer.AddTool(mcp.Tool{
		Name: "explain_code_with_context",
		Description: `Get explanation of code with relevant context from the entire codebase.

Use when:
- User asks "how does this work" or "explain this code"
- Need to understand code in relation to the rest of the system
- Finding dependencies, callers, or related implementations

Automatically retrieves relevant surrounding code for better understanding.`,
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "Path to file to explain",
				},
				"focus": map[string]interface{}{
					"type":        "string",
					"description": "Optional: specific aspect to focus on (e.g., 'dependencies', 'callers', 'implementation')",
				},
			},
			Required: []string{"file_path"},
		},
	}, s.handleExplainCode)

	// Index directory
	mcpServer.AddTool(mcp.Tool{
		Name: "index_codebase",
		Description: `Index a directory for semantic search. Run this FIRST before using semantic search.

Use when:
- Starting a new session with a codebase
- Code has been significantly updated
- Adding a new project directory

This builds the semantic search index. Takes 30s-2min depending on codebase size.`,
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Root path to index (e.g., '/Users/denis/projects/terraform-iac')",
				},
				"extensions": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
					"description": "File extensions to include (default: ['.go', '.py', '.js', '.ts', '.tf', '.yaml'])",
					"default":     []string{".go", ".py", ".js", ".ts", ".tf", ".yaml", ".yml"},
				},
			},
			Required: []string{"path"},
		},
	}, s.handleIndexDirectory)

	// Get index stats
	mcpServer.AddTool(mcp.Tool{
		Name: "get_index_stats",
		Description: `Get statistics about the current semantic search index.

Shows:
- Number of indexed files and chunks
- Supported languages
- Last index time
- Index size

Use to verify index is ready before searching.`,
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}, s.handleGetStats)

	// Get indexing progress
	mcpServer.AddTool(mcp.Tool{
		Name: "get_indexing_progress",
		Description: `Get real-time progress of background indexing.

Shows:
- Current status (in_progress, completed, failed)
- Number of files indexed vs total
- Progress percentage
- Failed files (if any)
- Estimated time remaining

Use this to monitor background indexing without blocking.`,
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}, s.handleGetIndexingProgress)

	// Re-index specific files (for git hooks)
	mcpServer.AddTool(mcp.Tool{
		Name: "reindex_files",
		Description: `Re-index specific files after modification (typically called by git hooks).

This tool:
1. Deletes old chunks for the specified files
2. Re-chunks and re-indexes the current content
3. Handles deleted files automatically

**Use cases:**
- After committing changes to files
- After pulling/merging code
- Manual re-indexing of specific files

**Note:** Files must be absolute paths.`,
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"file_paths": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
					"description": "List of absolute file paths to re-index",
				},
			},
			Required: []string{"file_paths"},
		},
	}, s.handleReindexFiles)
}
