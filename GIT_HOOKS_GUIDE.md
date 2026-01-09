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

### Automatic workflow

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
┌─────────────────┐
│  4. Create      │     .code-rag-pending-reindex
│  marker file    │     contains modified files
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  5. Auto        │     MCP server processes on startup
│  re-index       │     (automatic since v1.0.0)
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

## 🛠️ Usage

### Standard workflow

```bash
# 1. Modify files
vim api/src/middleware/auth.js
vim config.yaml

# 2. Commit (hook triggers automatically)
git add .
git commit -m "Update auth middleware"

# Output from hook:
# 🔍 code-rag: Detecting changed files...
# 📝 Re-indexing 2 file(s)...
# 📤 Files to re-index:
#    - auth.js
#    - config.yaml
# ✅ Re-index request queued (will be processed on next MCP server start)

# 3. Restart MCP server to process re-indexing
# The server will automatically detect and process pending files
```

### After a pull/merge

```bash
git pull origin main

# Output from post-merge hook:
# 🔍 code-rag: Detecting merged files...
# 📝 Re-indexing 5 file(s) from merge...
# ✅ Re-index request queued (will be processed on next MCP server start)
```

### Manual re-indexing

If you want to force re-indexing of specific files:

```bash
# Via the MCP reindex_files tool
# Arguments:
{
  "file_paths": [
    "/absolute/path/to/file1.js",
    "/absolute/path/to/file2.py"
  ]
}
```

## 📝 Marker file: `.code-rag-pending-reindex`

The hook creates this file at your project root with the list of files to re-index.

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

✅ **Automatic**: No need to think about re-indexing
✅ **Lightweight**: No always-running watcher
✅ **Reliable**: Git knows exactly what changed
✅ **Performant**: Only indexes modified files
✅ **Multi-developer**: Each dev re-indexes their own changes

## ✨ Features (since v1.0.0)

✅ **Automatic processing**: Marker file is processed on MCP server startup
✅ **No manual intervention**: Just restart the server and pending files are re-indexed
✅ **Error handling**: Failed re-indexing is logged without blocking startup

## 🔮 Future improvements

- [ ] REST API to call `reindex_files` without MCP
- [ ] Optional pre-push hook (blocking until indexing complete)
- [ ] Web dashboard to view ongoing re-indexing

## 💡 Tips

1. **Commit often**: Each commit only re-indexes what changed
2. **Use branches**: The global index stays consistent even with multiple branches
3. **Restart server**: The marker file is processed automatically on next startup

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
          # Call your MCP server or API
          curl -X POST https://your-mcp-server/reindex
```

---

**Questions?** Check the main README or open an issue.
