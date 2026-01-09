package config

import (
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	// Server
	ServerName    string
	ServerVersion string

	// Qdrant
	QdrantURL      string
	QdrantAPIKey   string
	CollectionName string

	// Embeddings
	EmbeddingType    string // "local", "lmstudio", or "openai"
	EmbeddingModel   string
	EmbeddingAPIKey  string
	EmbeddingBaseURL string // LM Studio URL
	EmbeddingDim     int

	// Indexing
	AutoIndexOnStartup bool
	CodePaths          []string
	FileExtensions     []string
	MaxFileSize        int64
	ChunkSize          int
	ChunkOverlap       int

	// Search
	TopK     int
	MinScore float32
}

func Load(configPath string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.config/code-rag-mcp")
		viper.AddConfigPath("/etc/code-rag-mcp")
	}

	// Defaults pour embeddings locaux
	viper.SetDefault("server_name", "code-rag")
	viper.SetDefault("server_version", "1.0.0")
	viper.SetDefault("qdrant_url", "localhost:6334")
	viper.SetDefault("collection_name", "code_embeddings")

	// Local embeddings par défaut
	viper.SetDefault("embedding_type", "local")
	viper.SetDefault("embedding_model", "nomic-ai/nomic-embed-text-v1.5-GGUF")
	viper.SetDefault("embedding_base_url", "http://localhost:1234/v1")
	viper.SetDefault("embedding_dim", 768) // nomic-embed default

	viper.SetDefault("auto_index_on_startup", false)
	viper.SetDefault("file_extensions", []string{".go", ".py", ".js", ".ts", ".tf", ".yaml", ".yml", ".md"})
	viper.SetDefault("max_file_size", 1024*1024)
	viper.SetDefault("chunk_size", 1000)
	viper.SetDefault("chunk_overlap", 200)
	viper.SetDefault("top_k", 5)
	viper.SetDefault("min_score", 0.7)

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	cfg := &Config{
		ServerName:         viper.GetString("server_name"),
		ServerVersion:      viper.GetString("server_version"),
		QdrantURL:          viper.GetString("qdrant_url"),
		QdrantAPIKey:       viper.GetString("qdrant_api_key"),
		CollectionName:     viper.GetString("collection_name"),
		EmbeddingType:      viper.GetString("embedding_type"),
		EmbeddingModel:     viper.GetString("embedding_model"),
		EmbeddingAPIKey:    viper.GetString("embedding_api_key"),
		EmbeddingBaseURL:   viper.GetString("embedding_base_url"),
		EmbeddingDim:       viper.GetInt("embedding_dim"),
		AutoIndexOnStartup: viper.GetBool("auto_index_on_startup"),
		CodePaths:          viper.GetStringSlice("code_paths"),
		FileExtensions:     viper.GetStringSlice("file_extensions"),
		MaxFileSize:        viper.GetInt64("max_file_size"),
		ChunkSize:          viper.GetInt("chunk_size"),
		ChunkOverlap:       viper.GetInt("chunk_overlap"),
		TopK:               viper.GetInt("top_k"),
		MinScore:           float32(viper.GetFloat64("min_score")),
	}

	// Override from env
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		cfg.EmbeddingAPIKey = apiKey
	}
	if lmStudioURL := os.Getenv("LM_STUDIO_URL"); lmStudioURL != "" {
		cfg.EmbeddingBaseURL = lmStudioURL
	}

	return cfg, nil
}
