#!/bin/bash
# Setup script for code-rag-mcp git hooks

set -e

echo "🪝 code-rag-mcp Git Hooks Setup"
echo "================================"
echo ""

# Check if we're in a git repository
if [ ! -d ".git" ]; then
  echo "❌ Error: Not a git repository"
  echo "   Run this script from your project root (where .git/ is located)"
  exit 1
fi

REPO_ROOT=$(git rev-parse --show-toplevel)
HOOKS_DIR="$REPO_ROOT/.git/hooks"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "📂 Repository: $REPO_ROOT"
echo "🔧 Hooks directory: $HOOKS_DIR"
echo ""

# Function to install a hook
install_hook() {
  local hook_name=$1
  local source_file="$SCRIPT_DIR/git-hooks/$hook_name"
  local target_file="$HOOKS_DIR/$hook_name"

  if [ ! -f "$source_file" ]; then
    echo "⚠️  Hook template not found: $source_file"
    return 1
  fi

  # Backup existing hook if present
  if [ -f "$target_file" ]; then
    echo "📦 Backing up existing $hook_name to ${hook_name}.backup"
    cp "$target_file" "$target_file.backup"
  fi

  # Copy and make executable
  cp "$source_file" "$target_file"
  chmod +x "$target_file"

  echo "✅ Installed: $hook_name"
}

echo "Installing hooks..."
echo ""

# Install post-commit hook
install_hook "post-commit"

# Install post-merge hook
install_hook "post-merge"

echo ""
echo "🎉 Git hooks installed successfully!"
echo ""
echo "📝 Installed hooks:"
echo "   - post-commit: Re-indexes files after each commit"
echo "   - post-merge:  Re-indexes files after pull/merge"
echo ""
echo "💡 How it works:"
echo "   1. You commit/merge code changes"
echo "   2. Hook detects modified files"
echo "   3. Creates .code-rag-pending-reindex marker"
echo "   4. MCP server processes re-indexing on next startup/check"
echo ""
echo "⚙️  Configuration:"
echo "   Edit config.yaml to adjust:"
echo "   - file_extensions: Which files to index"
echo "   - code_paths: Which directories to watch"
echo ""
echo "✨ Ready! Try making a commit to test it out."
