# Contributing to Code RAG MCP Server

Thank you for your interest in contributing to Code RAG MCP Server! We welcome contributions from the community.

## How to Contribute

### Reporting Bugs

If you find a bug, please open an issue with:
- A clear description of the problem
- Steps to reproduce
- Expected vs actual behavior
- Your environment (OS, Go version, etc.)
- Relevant logs or error messages

### Suggesting Features

Feature requests are welcome! Please open an issue describing:
- The feature you'd like to see
- Why it would be useful
- Potential implementation approach (optional)

### Pull Requests

1. **Fork the repository** and create your branch from `main`:
   ```bash
   git checkout -b feature/my-feature
   ```

2. **Make your changes**:
   - Write clear, documented code
   - Follow Go best practices and conventions
   - Add tests for new functionality
   - Update documentation as needed

3. **Test your changes**:
   ```bash
   # Run tests
   go test ./...

   # Build the project
   make build

   # Test manually
   ./code-rag-mcp
   ```

4. **Commit your changes**:
   - Use clear, descriptive commit messages
   - Reference related issues (e.g., "Fixes #123")

5. **Push to your fork** and submit a pull request

## Development Setup

### Prerequisites

- Go 1.22 or later
- Docker (for Qdrant)
- LM Studio (for local embeddings)

### Setup Instructions

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/code-rag-mcp.git
cd code-rag-mcp

# Install dependencies
go mod download

# Build
make build

# Run tests
go test ./...

# Start Qdrant
docker run -d --name qdrant \
  -p 6333:6333 -p 6334:6334 \
  qdrant/qdrant
```

## Code Style

- Follow standard Go formatting (`gofmt`)
- Use meaningful variable and function names
- Add comments for exported functions and complex logic
- Keep functions focused and modular
- Error handling should be explicit and informative

## Testing

- Write unit tests for new functionality
- Ensure existing tests pass
- Test with both local and OpenAI embeddings when applicable
- Test MCP integration with Claude or Zed

## Documentation

- Update README.md for user-facing changes
- Update code comments for API changes
- Add examples for new features
- Update CHANGELOG.md with your changes

## Git Hooks

This project includes git hooks for automatic re-indexing. To set them up:

```bash
./setup-git-hooks.sh
```

## Questions?

Feel free to open an issue for any questions about contributing!

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
