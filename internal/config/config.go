package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	SupabaseURL    string `yaml:"supabase_url"`
	SupabaseKey    string `yaml:"supabase_key"`
	OpenRouterKey  string `yaml:"openrouter_key"`
	OpenRouterBase string `yaml:"openrouter_base"`
	EmbeddingModel string `yaml:"embedding_model"`
	InferenceModel string `yaml:"inference_model"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
