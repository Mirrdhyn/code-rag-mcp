#!/bin/bash

set -e

echo "🚀 Code RAG MCP Server - Installation"
echo "======================================"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check prerequisites
echo "📋 Checking prerequisites..."

# Check Go
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed${NC}"
    echo "Install Go: brew install go"
    exit 1
fi
echo -e "${GREEN}✅ Go installed: $(go version)${NC}"

# Check Docker
if ! command -v docker &> /dev/null; then
    echo -e "${YELLOW}⚠️  Docker not found (needed for Qdrant)${NC}"
    echo "Install Docker: brew install docker"
fi

echo ""
echo "🔨 Building..."
go build -o code-rag-mcp .

echo ""
echo "📦 Installing to /usr/local/bin..."
sudo cp code-rag-mcp /usr/local/bin/

echo ""
echo "📝 Setting up configuration..."
CONFIG_DIR="$HOME/.config/code-rag-mcp"
mkdir -p "$CONFIG_DIR"

if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
    cp config.yaml "$CONFIG_DIR/"
    echo -e "${GREEN}✅ Config copied to $CONFIG_DIR/config.yaml${NC}"
    echo -e "${YELLOW}⚠️  Please edit $CONFIG_DIR/config.yaml with your code paths${NC}"
else
    echo -e "${YELLOW}⚠️  Config already exists, skipping...${NC}"
fi

echo ""
echo "🐳 Setting up Qdrant..."
if docker ps -a | grep -q qdrant; then
    echo "Qdrant container already exists"
    docker start qdrant || true
else
    docker run -d --name qdrant \
        -p 6333:6333 -p 6334:6334 \
        -v "$(pwd)/qdrant_data:/qdrant/storage" \
        qdrant/qdrant
fi
echo -e "${GREEN}✅ Qdrant started${NC}"

echo ""
echo "================================================"
echo -e "${GREEN}✅ Installation complete!${NC}"
echo "================================================"
echo ""
echo "Next steps:"
echo ""
echo "1. Install LM Studio:"
echo "   https://lmstudio.ai"
echo ""
echo "2. Download embedding model in LM Studio:"
echo "   - Search: 'nomic-embed-text'"
echo "   - Download: nomic-ai/nomic-embed-text-v1.5-GGUF (Q8)"
echo "   - Start Local Server on port 1234"
echo ""
echo "3. Edit config with your code paths:"
echo "   nano $CONFIG_DIR/config.yaml"
echo ""
echo "4. Configure Claude Desktop:"
echo "   cp examples/claude_desktop_config.json ~/Library/Application\\ Support/Claude/"
echo ""
echo "5. Test the server:"
echo "   code-rag-mcp"
echo ""
echo "Documentation: README.md"
