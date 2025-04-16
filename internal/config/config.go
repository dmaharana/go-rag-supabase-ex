package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Database DbConfig  `yaml:"database"`
	EmbedLLM LLMConfig `yaml:"embed_llm"`
	QueryLLM LLMConfig `yaml:"query_llm"`
}

type DbConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type LLMConfig struct {
	BaseURL string `yaml:"llm_base_url"`
	Key     string `yaml:"llm_key"`
	Model   string `yaml:"llm_model"`
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
