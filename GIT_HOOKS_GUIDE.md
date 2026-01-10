# 🪝 Git Hooks Guide - Auto Re-indexing

This guide explains how to use the automatic re-indexing system based on Git hooks.

## 📋 Overview

The system uses **Git hooks** to detect modified files and re-index them automatically after each commit or merge. This ensures that your semantic search index always stays in sync with your code.

## 🚀 Installation

### 1. Install hooks in your project

From your project directory (where `.git/` is located):

```bash
/path/to/code-rag-mcp/setup-git-hooks.sh
```

**Example:**
```bash
cd /Users/you/projects/my-project
/path/to/code-rag-mcp/setup-git-hooks.sh
```

### 2. Verify installation

```bash
ls -la .git/hooks/post-*
# Should display:
# post-commit
# post-merge
```

## 🔄 How it works

### Automatic workflow (v1.1.0+)

```
┌─────────────────┐
│  1. Modify      │
│  auth.js        │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  2. Commit      │
│  git commit     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐     🎣 Post-commit hook
│  3. Automatic   │     triggers
│  detection      │
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────────────────┐
│  4. HTTP API call                           │
│  POST http://localhost:9333/reindex         │
│  (immediate re-indexing)                    │
└────────┬────────────────────────────────────┘
         │
         │ If API unavailable (fallback):
         ▼
┌─────────────────┐
│  5. Create      │     .code-rag-pending-reindex
│  marker file    │     (processed on next startup)
└─────────────────┘
```

### Tracked files

The hook automatically detects changes to:
- `.js`, `.jsx`, `.ts`, `.tsx` (JavaScript/TypeScript)
- `.go` (Go)
- `.py` (Python)
- `.md` (Markdown)
- `.tf` (Terraform)
- `.yaml`, `.yml` (YAML)
- `.json` (JSON)
- `.sh` (Shell)

### Deleted files

When you delete a file, the hook:
1. Detects the deletion
2. Removes corresponding chunks from the index
3. Doesn't attempt to re-index the deleted file

## 🌐 HTTP API (v1.1.0+)

The MCP server now includes an HTTP API for immediate re-indexing without waiting for server restart.

### Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/reindex` | POST | Re-index specific files |
| `/reindex-pending` | POST | Process marker file |

### Configuration

In `config.yaml`:

```yaml
# HTTP API configuration
http_api_enabled: true
http_api_port: 9333
```

Or via environment variables:
```bash
export CODE_RAG_HTTP_PORT=9333
export CODE_RAG_HTTP_HOST=localhost
```

### Usage Examples

**Re-index specific files:**
```bash
curl -X POST http://localhost:9333/reindex \
  -H "Content-Type: application/json" \
  -d '{"files": ["/path/to/file1.js", "/path/to/file2.py"]}'
```

**Process pending marker file:**
```bash
curl -X POST "http://localhost:9333/reindex-pending?workdir=/path/to/project"
```

**Health check:**
```bash
curl http://localhost:9333/health
# {"status":"ok","version":"1.1.0"}
```

## 🛠️ Usage

### Standard workflow

```bash
# 1. Modify files
vim api/src/middleware/auth.js
vim config.yaml

# 2. Commit (hook triggers automatically)
git add .
git commit -m "Update auth middleware"

# Output from hook (if API available):
# 🔍 code-rag: Detecting changed files...
# 📝 2 file(s) to re-index...
# 📤 Files to re-index:
#    - auth.js
#    - config.yaml
# 🔄 Calling code-rag HTTP API for immediate re-indexing...
# ✅ Re-indexed 2 file(s) successfully

# Output if API not available:
# ⚠️  code-rag HTTP API not available at http://localhost:9333/reindex
# 📋 Re-index request queued in marker file
```

### After a pull/merge

```bash
git pull origin main

# Output from post-merge hook:
# 🔍 code-rag: Detecting merged files...
# 📝 5 file(s) to re-index...
# 🔄 Calling code-rag HTTP API for immediate re-indexing...
# ✅ Re-indexed 5 file(s) successfully
```

### Manual re-indexing

If you want to force re-indexing of specific files:

**Via HTTP API:**
```bash
curl -X POST http://localhost:9333/reindex \
  -H "Content-Type: application/json" \
  -d '{"files": ["/absolute/path/to/file1.js", "/absolute/path/to/file2.py"]}'
```

**Via MCP tool:**
```json
{
  "file_paths": [
    "/absolute/path/to/file1.js",
    "/absolute/path/to/file2.py"
  ]
}
```

## 📝 Marker file: `.code-rag-pending-reindex`

When the HTTP API is not available, the hook creates this file at your project root with the list of files to re-index.

**Format:**
```
/absolute/path/to/file1.js /absolute/path/to/file2.py /absolute/path/to/file3.md
```

**Note:** This file is automatically added to `.gitignore`

## 🔧 Configuration

### Customize tracked extensions

Edit the hooks in `.git/hooks/post-commit` and `.git/hooks/post-merge`:

```bash
# Modify this line:
EXTENSIONS="\.js$|\.jsx$|\.ts$|\.tsx$|\.go$|\.py$|\.md$|\.tf$|\.yaml$|\.yml$|\.json$"

# Example: add Rust (.rs)
EXTENSIONS="\.js$|\.jsx$|\.ts$|\.tsx$|\.go$|\.py$|\.md$|\.tf$|\.rs$"
```

### Customize HTTP API port

```bash
# In your shell profile (~/.bashrc, ~/.zshrc)
export CODE_RAG_HTTP_PORT=9333
export CODE_RAG_HTTP_HOST=localhost
```

### Temporarily disable

```bash
# Rename the hook
mv .git/hooks/post-commit .git/hooks/post-commit.disabled

# Re-enable
mv .git/hooks/post-commit.disabled .git/hooks/post-commit
```

## 🐛 Troubleshooting

### Hook doesn't trigger

1. **Check permissions**:
   ```bash
   chmod +x .git/hooks/post-commit
   chmod +x .git/hooks/post-merge
   ```

2. **Verify you're in a Git repo**:
   ```bash
   git status
   ```

3. **Check hook content**:
   ```bash
   cat .git/hooks/post-commit
   ```

### HTTP API not available

1. **Check if MCP server is running** with HTTP API enabled
2. **Verify port**:
   ```bash
   curl http://localhost:9333/health
   ```
3. **Check config.yaml**:
   ```yaml
   http_api_enabled: true
   http_api_port: 9333
   ```

### No files detected

The hook displays "No code files changed" if:
- Modified files don't match tracked extensions
- You only modified excluded files (e.g., `.gitignore`, images)

### Test hook manually

```bash
# Simulate a commit
.git/hooks/post-commit

# Should display file detection
```

## 📊 Benefits of this approach

✅ **Immediate**: HTTP API provides instant re-indexing (no restart needed)
✅ **Automatic**: No need to think about re-indexing
✅ **Reliable**: Git knows exactly what changed
✅ **Performant**: Only indexes modified files
✅ **Fallback**: Marker file ensures no changes are lost

## ✨ Features

| Version | Feature |
|---------|---------|
| v1.0.0 | Marker file processed on startup |
| v1.1.0 | HTTP API for immediate re-indexing |

## 🤝 CI/CD Integration

If you have a build server, you can add a step to re-index automatically:

```yaml
# .github/workflows/index.yml
name: Re-index on push
on: [push]
jobs:
  reindex:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Re-index changed files
        run: |
          curl -X POST http://your-mcp-server:9333/reindex \
            -H "Content-Type: application/json" \
            -d '{"files": ${{ steps.changed-files.outputs.all_changed_files }}}'
```

---

**Questions?** Check the main README or open an issue.
