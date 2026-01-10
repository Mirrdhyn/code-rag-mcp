# Code RAG MCP Server

MCP (Model Context Protocol) server for semantic code search. Replace grep/ripgrep with understanding-based code search.

## 🚀 Features

- ✅ **Semantic Search**: Find code by concept, not just by text
- ✅ **Local Embeddings**: Uses LM Studio (no OpenAI required)
- ✅ **Multi-language**: Go, Python, JS/TS, Terraform, YAML, etc.
- ✅ **MCP Integration**: Compatible with Claude Code and Zed
- ✅ **HTTP API**: Git hooks for automatic re-indexing on commit
- ✅ **Fast**: In-memory indexing with Qdrant

## 📋 Prerequisites

1. **Go 1.22+**
   ```bash
   brew install go
   ```

2. **Qdrant** (Vector Database)
   ```bash
   docker run -d --name qdrant \
     -p 6333:6333 -p 6334:6334 \
     -v $(pwd)/qdrant_data:/qdrant/storage \
     qdrant/qdrant
   ```

3. **LM Studio** with an embedding model
   - Download: https://lmstudio.ai
   - Load model: `nomic-ai/nomic-embed-text-v1.5-GGUF`
   - Start local server on port 1234

## 🔧 Installation

```bash
# 1. Clone the repo
git clone <your-repo>
cd code-rag-mcp

# 2. Install dependencies
go mod download

# 3. Build
go build -o code-rag-mcp

# 4. Configure
cp config.yaml ~/.config/code-rag-mcp/config.yaml
# Edit config.yaml with your code paths

# 5. Install globally (optional)
sudo cp code-rag-mcp /usr/local/bin/
```

## ⚙️ Configuration

### 1. Config.yaml

Edit `~/.config/code-rag-mcp/config.yaml`:

```yaml
embedding_type: "local"
embedding_model: "nomic-ai/nomic-embed-text-v1.5-GGUF"
embedding_base_url: "http://localhost:1234/v1"
embedding_dim: 768

code_paths:
  - "/Users/you/projects"  # Your code directory
```

### 2. Claude Desktop

**File**: `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS)
**File**: `~/.config/Claude/claude_desktop_config.json` (Linux)

```json
{
  "mcpServers": {
    "code-rag": {
      "command": "/usr/local/bin/code-rag-mcp",
      "env": {
        "LM_STUDIO_URL": "http://localhost:1234/v1"
      }
    }
  }
}
```

After modification, **fully restart Claude Desktop** (Quit + relaunch).

### 3. Claude Code

**Option A: Via CLI (Recommended)** 🚀

```bash
# Add the code-rag MCP server
claude mcp add code-rag \
  --type stdio \
  --command /usr/local/bin/code-rag-mcp \
  --scope user

# Verify installation
claude mcp list
claude mcp get code-rag
```

**Option B: Manual configuration**

Claude Code uses the **same file** as Claude Desktop:
- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Linux: `~/.config/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "code-rag": {
      "command": "/usr/local/bin/code-rag-mcp",
      "env": {
        "LM_STUDIO_URL": "http://localhost:1234/v1"
      }
    }
  }
}
```

**Complete copy-paste command** (after building):

```bash
# 1. Install the binary
sudo cp code-rag-mcp /usr/local/bin/
sudo chmod +x /usr/local/bin/code-rag-mcp

# 2. Add to Claude Code
claude mcp add code-rag \
  --type stdio \
  --command /usr/local/bin/code-rag-mcp \
  --scope user \
  --env LM_STUDIO_URL=http://localhost:1234/v1

# 3. Verify
claude mcp list
```

### 4. Zed

**File**: `~/.config/zed/settings.json`

```json
{
  "context_servers": {
    "code-rag": {
      "source": "custom",
      "command": {
        "path": "/usr/local/bin/code-rag-mcp",
        "args": [],
        "env": {
          "LM_STUDIO_URL": "http://localhost:1234/v1"
        }
      }
    }
  }
}
```

**Complete copy-paste command**:

```bash
# 1. Install the binary (if not already done)
sudo cp code-rag-mcp /usr/local/bin/
sudo chmod +x /usr/local/bin/code-rag-mcp

# 2. Add configuration to Zed
cat << 'EOF' >> ~/.config/zed/settings.json
{
  "context_servers": {
    "code-rag": {
      "source": "custom",
      "command": {
        "path": "/usr/local/bin/code-rag-mcp",
        "args": [],
        "env": {
          "LM_STUDIO_URL": "http://localhost:1234/v1"
        }
      }
    }
  }
}
EOF

# 3. Restart Zed
```

> **Note**: For Zed, you can also use the "Add Custom Server" button in the Agent Panel.

## 🎯 Usage

### 1. Start services

```bash
# Terminal 1: Qdrant
docker start qdrant

# Terminal 2: LM Studio
# Open LM Studio UI and start the server

# Terminal 3: Test the server
./code-rag-mcp
```

### 2. Use in Claude

```
Claude: Hi! Let me check the index status.
[Calls: get_index_stats]

You: Index my Terraform project
Claude: [Calls: index_codebase /path/to/terraform]

You: Where is the VPC configuration?
Claude: [Calls: semantic_code_search "VPC network configuration"]
```

### 3. Example queries

**Semantic search:**
```
"authentication middleware"
"database connection logic"
"error handling patterns"
"terraform AWS VPC modules"
"API rate limiting"
```

**Similar code:**
```
"Find code similar to: func (s *Server) HandleAuth()"
```

**Explanation with context:**
```
"Explain handlers/auth.go with related dependencies"
```

## 🔍 Available MCP Tools

### `semantic_code_search`
Primary semantic search. **Use instead of grep.**

```json
{
  "query": "authentication logic",
  "limit": 5,
  "min_score": 0.7,
  "language": "go"
}
```

### `find_similar_code`
Find code similar to a given snippet.

```json
{
  "code_snippet": "func HandleError(err error) { ... }",
  "limit": 5
}
```

### `explain_code_with_context`
Explain a file with its context.

```json
{
  "file_path": "/path/to/file.go",
  "focus": "dependencies"
}
```

### `index_codebase`
Index a directory. **Run this first.**

```json
{
  "path": "/Users/you/projects/myapp",
  "extensions": [".go", ".py"]
}
```

### `get_index_stats`
Check index status.

## 🧪 Tests

```bash
# Test embeddings
go run scripts/test_embeddings.go

# Test indexing
go run scripts/test_indexing.go /path/to/code

# Test searches
go run scripts/test_search.go "authentication"
```

## 📊 Recommended Embedding Models

| Model | RAM | Dim | Quality | Usage |
|--------|-----|-----|---------|-------|
| **nomic-embed-v1.5** (Q8) | 548MB | 768 | ⭐⭐⭐⭐⭐ | **Recommended** |
| bge-small-en-v1.5 | 133MB | 384 | ⭐⭐⭐⭐ | Lightweight |
| all-MiniLM-L6-v2 | 90MB | 384 | ⭐⭐⭐ | Fast |

### Download in LM Studio

1. Open LM Studio
2. Search: "nomic-embed-text"
3. Download: `nomic-ai/nomic-embed-text-v1.5-GGUF` (Q8)
4. Load the model in "Local Server"

## 🐛 Troubleshooting

### No results
```bash
# Check index
get_index_stats

# Re-index
index_codebase /path/to/code

# Lower minimum score
semantic_code_search "query" min_score=0.5
```

### LM Studio connection fails
```bash
# Check if LM Studio is running
curl http://localhost:1234/v1/models

# Check logs
./code-rag-mcp --config config.yaml
```

### Qdrant not responding
```bash
# Restart Qdrant
docker restart qdrant

# Check logs
docker logs qdrant
```

## 📚 Documentation

- [Instructions for Claude](docs/CLAUDE_INSTRUCTIONS.md)
- [Integration Guide](docs/INTEGRATION.md)
- [API Reference](docs/API.md)

## 🤝 Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md)

## 📄 License

MIT License - see [LICENSE](LICENSE)

## 🙏 Acknowledgments

- [MCP Protocol](https://github.com/mark3labs/mcp-go)
- [Qdrant](https://qdrant.tech)
- [LM Studio](https://lmstudio.ai)
- [Nomic AI](https://www.nomic.ai)
