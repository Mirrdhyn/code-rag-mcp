# Integration Guide

Guide d'intégration du serveur MCP Code RAG avec différents outils.

## Claude Desktop

### Configuration

Fichier: `~/Library/Application Support/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "code-rag": {
      "command": "/usr/local/bin/code-rag-mcp",
      "args": [],
      "env": {
        "LM_STUDIO_URL": "http://localhost:1234/v1",
        "CONFIG_PATH": "/Users/denis/.config/code-rag-mcp/config.yaml"
      }
    }
  }
}
```

### Vérification

1. Fermer complètement Claude Desktop
2. Rouvrir Claude Desktop
3. Dans une nouvelle conversation, taper: "Can you see the code-rag tools?"
4. Claude devrait lister les outils disponibles

### Utilisation

```
You: Index my project at /Users/denis/projects/myapp

Claude: [Uses index_codebase tool]

You: Find all authentication logic

Claude: [Uses semantic_code_search]
```

## Claude Code (CLI)

### Configuration

Fichier: `~/.config/claude/config.json`

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

### Utilisation

```bash
$ claude code

# Première utilisation
You: Index the current directory
Claude: [Indexes codebase]

# Recherche
You: Where is the database connection code?
Claude: [Uses semantic search]
```

## Zed Editor

### Configuration

Fichier: `~/.config/zed/settings.json`

```json
{
  "assistant": {
    "version": "2",
    "default_model": {
      "provider": "anthropic",
      "model": "claude-sonnet-4-20250514"
    }
  },
  "context_servers": {
    "code-rag": {
      "command": {
        "path": "/usr/local/bin/code-rag-mcp",
        "env": {
          "LM_STUDIO_URL": "http://localhost:1234/v1"
        }
      }
    }
  }
}
```

### Utilisation

1. Ouvrir Zed
2. Cmd+Shift+A pour l'assistant
3. L'assistant a accès aux outils MCP automatiquement

## API directe (pour développeurs)

### Démarrer le serveur

```bash
code-rag-mcp
```

Le serveur MCP écoute sur stdin/stdout selon le protocole MCP.

### Exemple d'appel (via mcp-client)

```javascript
const client = new MCPClient();
await client.connect('/usr/local/bin/code-rag-mcp');

// Recherche sémantique
const results = await client.callTool('semantic_code_search', {
  query: 'authentication middleware',
  limit: 5,
  min_score: 0.7
});

console.log(results);
```

## Variables d'environnement

Toutes les configurations peuvent être surchargées via variables d'environnement:

```bash
export LM_STUDIO_URL="http://localhost:1234/v1"
export QDRANT_URL="localhost:6334"
export CONFIG_PATH="/custom/path/config.yaml"

code-rag-mcp
```

## Configurations multiples

### Profile "work"

`~/.config/code-rag-mcp/config.work.yaml`:
```yaml
code_paths:
  - "/Users/denis/work"
embedding_type: "local"
```

### Profile "personal"

`~/.config/code-rag-mcp/config.personal.yaml`:
```yaml
code_paths:
  - "/Users/denis/projects"
embedding_type: "openai"
embedding_api_key: "sk-..."
```

### Usage

```bash
# Work
code-rag-mcp --config ~/.config/code-rag-mcp/config.work.yaml

# Personal
code-rag-mcp --config ~/.config/code-rag-mcp/config.personal.yaml
```

## Systemd Service (Linux)

`/etc/systemd/system/code-rag-mcp.service`:

```ini
[Unit]
Description=Code RAG MCP Server
After=network.target docker.service

[Service]
Type=simple
User=denis
ExecStart=/usr/local/bin/code-rag-mcp
Restart=on-failure
Environment="LM_STUDIO_URL=http://localhost:1234/v1"

[Install]
WantedBy=multi-user.target
```

Activer:
```bash
sudo systemctl enable code-rag-mcp
sudo systemctl start code-rag-mcp
```

## Launchd (macOS)

`~/Library/LaunchAgents/com.coderag.mcp.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.coderag.mcp</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/code-rag-mcp</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>EnvironmentVariables</key>
    <dict>
        <key>LM_STUDIO_URL</key>
        <string>http://localhost:1234/v1</string>
    </dict>
</dict>
</plist>
```

Activer:
```bash
launchctl load ~/Library/LaunchAgents/com.coderag.mcp.plist
```

## Monitoring

### Logs

```bash
# Avec systemd
journalctl -u code-rag-mcp -f

# Avec launchd
log stream --predicate 'process == "code-rag-mcp"'

# Direct
code-rag-mcp 2>&1 | tee code-rag.log
```

### Health check

```bash
# Script simple
#!/bin/bash
if pgrep -f code-rag-mcp > /dev/null; then
    echo "✅ Server running"
else
    echo "❌ Server not running"
    exit 1
fi
```

## Sécurité

### Recommandations

1. **Ne jamais exposer le serveur MCP directement sur le réseau**
   - MCP est conçu pour stdin/stdout uniquement
   - Utiliser via Claude Desktop/Code/Zed uniquement

2. **Protéger les configs**
   ```bash
   chmod 600 ~/.config/code-rag-mcp/config.yaml
   ```

3. **API Keys**
   - Utiliser des variables d'environnement
   - Ne jamais committer les configs avec des clés

4. **Qdrant**
   - Si exposé, utiliser authentication
   - Firewall pour limiter l'accès local uniquement

## Performance

### Optimisations

1. **Indexation initiale**
   - Indexer pendant les heures creuses
   - Utiliser un SSD pour Qdrant

2. **Embeddings**
   - nomic-embed Q8: meilleur compromis
   - Q4 si RAM limitée
   - OpenAI si besoin de vitesse maximale

3. **Recherche**
   - Commencer avec min_score=0.7
   - Ajuster selon les résultats
   - Limiter les résultats (5-10 suffisent)

## Troubleshooting

Voir [README.md](../README.md) section Troubleshooting.
