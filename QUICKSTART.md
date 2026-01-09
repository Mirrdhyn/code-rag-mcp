# 🚀 Quick Start Guide

Installation and setup in 10 minutes!

## Step 1: Install prerequisites

```bash
# macOS
brew install go
brew install docker
brew install --cask lm-studio
```

## Step 2: Automated installation

```bash
# Clone and install
git clone <your-repo>
cd code-rag-mcp
./install.sh
```

This will:
- ✅ Compile the binary
- ✅ Install it in `/usr/local/bin`
- ✅ Create config in `~/.config/code-rag-mcp`
- ✅ Start Qdrant in Docker

## Step 3: Configure LM Studio

1. **Open LM Studio**

2. **Download the embedding model:**
   - Go to "Search" tab
   - Search: `nomic-embed-text`
   - Download: `nomic-ai/nomic-embed-text-v1.5-GGUF` **(Q8 recommended)**

3. **Start the server:**
   - Go to "Local Server" tab
   - Load the nomic-embed model
   - Port: 1234
   - Click "Start Server"

4. **Verify:**
   ```bash
   curl http://localhost:1234/v1/models
   ```

## Step 4: Configure your code paths

Edit `~/.config/code-rag-mcp/config.yaml`:

```yaml
code_paths:
  - "/Users/you/projects"           # ← Your main directory
  - "/Users/you/work/terraform"     # ← Other projects
```

## Step 5: Configure Claude

### For Claude Desktop

Copy the config:
```bash
cp examples/claude_desktop_config.json \
   ~/Library/Application\ Support/Claude/claude_desktop_config.json
```

Or manually create `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "code-rag": {
      "command": "/usr/local/bin/code-rag-mcp"
    }
  }
}
```

**Restart Claude Desktop**

### For Claude Code

```bash
mkdir -p ~/.config/claude
cp examples/claude_code_config.json ~/.config/claude/config.json
```

### For Zed

```bash
cp examples/zed_settings.json ~/.config/zed/settings.json
```

## Step 6: First test

```bash
# Check that everything works
code-rag-mcp
```

You should see:
```
INFO    Starting Code RAG MCP Server
INFO    Embedder initialized successfully  dimension=768
INFO    MCP Server starting...
```

## Step 7: Use in Claude

Open Claude Desktop and test:

```
You: Check if the index is ready

Claude: [Calls get_index_stats]
        The index is empty. Would you like me to index your codebase?

You: Yes, index /Users/you/projects/myapp

Claude: [Calls index_codebase]
        Indexing... this will take about 1 minute.
        ✅ Successfully indexed 1,247 code chunks!

You: Where is the authentication logic?

Claude: [Calls semantic_code_search "authentication logic"]
        Found authentication in 3 main areas:
        1. auth/middleware.go - JWT validation
        2. handlers/auth.go - Login endpoint
        3. services/auth.go - Core logic
```

## Quick troubleshooting

### Problem: "Failed to connect to LM Studio"
```bash
# Check if LM Studio is running
curl http://localhost:1234/v1/models

# If it doesn't work:
# 1. Open LM Studio
# 2. Load the nomic-embed model
# 3. Start "Local Server"
```

### Problem: "Failed to connect to Qdrant"
```bash
# Restart Qdrant
docker restart qdrant

# Or relaunch
docker run -d --name qdrant \
  -p 6333:6333 -p 6334:6334 \
  qdrant/qdrant
```

### Problem: "No results found"
```bash
# Check the index
code-rag-mcp

# In Claude:
get_index_stats

# Re-index if needed
index_codebase /path/to/your/code
```

## Useful commands

```bash
# Start Qdrant
docker start qdrant

# Stop Qdrant
docker stop qdrant

# View logs
docker logs qdrant

# Test embeddings
go run scripts/test_embeddings.go

# Rebuild
make build

# Clean everything and start over
make clean
make setup
```

## Support

- Full documentation: [README.md](README.md)
- Claude guide: [docs/CLAUDE_INSTRUCTIONS.md](docs/CLAUDE_INSTRUCTIONS.md)
- Issues: <your-repo>/issues

## That's it! 🎉

You can now use semantic search in Claude!
