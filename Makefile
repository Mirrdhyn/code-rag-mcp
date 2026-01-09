.PHONY: build run test clean install docker-qdrant setup help

# Variables
BINARY_NAME=code-rag-mcp
INSTALL_PATH=/usr/local/bin
CONFIG_DIR=$(HOME)/.config/code-rag-mcp

help: ## Afficher l'aide
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Compiler le binaire
	@echo "🔨 Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) .
	@echo "✅ Build complete: ./$(BINARY_NAME)"

run: build ## Compiler et lancer
	@echo "🚀 Starting $(BINARY_NAME)..."
	@./$(BINARY_NAME)

test: ## Lancer les tests
	@echo "🧪 Running tests..."
	@go test -v ./...

test-embeddings: build ## Tester les embeddings
	@echo "🧪 Testing embeddings..."
	@go run scripts/test_embeddings.go

test-search: build ## Tester la recherche
	@echo "🔍 Testing search..."
	@go run scripts/test_search.go "authentication"

clean: ## Nettoyer les fichiers générés
	@echo "🧹 Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -rf qdrant_data
	@echo "✅ Clean complete"

install: build ## Installer globalement
	@echo "📦 Installing to $(INSTALL_PATH)..."
	@sudo cp $(BINARY_NAME) $(INSTALL_PATH)/
	@mkdir -p $(CONFIG_DIR)
	@if [ ! -f $(CONFIG_DIR)/config.yaml ]; then \
		cp config.yaml $(CONFIG_DIR)/; \
		echo "📝 Config copied to $(CONFIG_DIR)/config.yaml"; \
	fi
	@echo "✅ Installation complete"

uninstall: ## Désinstaller
	@echo "🗑️  Uninstalling..."
	@sudo rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "✅ Uninstall complete"

docker-qdrant: ## Démarrer Qdrant avec Docker
	@echo "🐳 Starting Qdrant..."
	@docker run -d --name qdrant \
		-p 6333:6333 -p 6334:6334 \
		-v $(PWD)/qdrant_data:/qdrant/storage \
		qdrant/qdrant
	@echo "✅ Qdrant started on ports 6333 (HTTP) and 6334 (gRPC)"

docker-stop: ## Arrêter Qdrant
	@echo "🛑 Stopping Qdrant..."
	@docker stop qdrant
	@echo "✅ Qdrant stopped"

docker-clean: ## Supprimer le container Qdrant
	@echo "🧹 Cleaning Qdrant container..."
	@docker stop qdrant || true
	@docker rm qdrant || true
	@echo "✅ Qdrant container removed"

setup: docker-qdrant ## Setup complet (Qdrant + Build + Install)
	@echo "⚙️  Running full setup..."
	@sleep 2
	@make build
	@make install
	@echo ""
	@echo "✅ Setup complete!"
	@echo ""
	@echo "Next steps:"
	@echo "1. Install and start LM Studio with nomic-embed model"
	@echo "2. Edit $(CONFIG_DIR)/config.yaml with your code paths"
	@echo "3. Run: $(BINARY_NAME)"

dev: docker-qdrant ## Mode développement
	@echo "🔧 Starting development environment..."
	@sleep 2
	@make run

lint: ## Linter le code
	@echo "🔍 Linting..."
	@golangci-lint run ./...

fmt: ## Formater le code
	@echo "✨ Formatting code..."
	@go fmt ./...

deps: ## Installer les dépendances
	@echo "📦 Installing dependencies..."
	@go mod download
	@go mod tidy
	@echo "✅ Dependencies installed"

update-deps: ## Mettre à jour les dépendances
	@echo "🔄 Updating dependencies..."
	@go get -u ./...
	@go mod tidy
	@echo "✅ Dependencies updated"
