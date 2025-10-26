package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the application's configuration model.
// It captures credentials, interests, filters, and engagement strategy.
type Config struct {
	Account     AccountConfig     `yaml:"account"`
	Credentials CredentialsConfig `yaml:"credentials"`
	Interests   InterestsConfig   `yaml:"interests"`
	Filters     FiltersConfig     `yaml:"filters"`
	Engagement  EngagementConfig  `yaml:"engagement"`
	LLM         LLMConfig         `yaml:"llm"`
}

type AccountConfig struct {
	Username string `yaml:"username"`
}

type CredentialsConfig struct {
	// X/Twitter API bearer token. If empty, read from env X_BEARER_TOKEN
	BearerToken string `yaml:"bearerToken"`
}

type InterestsConfig struct {
	// Topics and keywords that define what we care about
	Topics   []string          `yaml:"topics"`
	Keywords []string          `yaml:"keywords"`
	Weights  map[string]float64 `yaml:"weights"` // optional per-keyword weight
}

type FiltersConfig struct {
	// Minimum acceptable organic content score [0,1]
	MinOrganicScore float64 `yaml:"minOrganicScore"`
	// Maximum acceptable bot likelihood [0,1]
	MaxBotLikelihood float64 `yaml:"maxBotLikelihood"`
	// Language filters, e.g., ["en"]
	Languages []string `yaml:"languages"`
}

type EngagementConfig struct {
	// Max interactions per hour and per day
	MaxPerHour int `yaml:"maxPerHour"`
	MaxPerDay  int `yaml:"maxPerDay"`
	// Quiet hours (UTC) to avoid low-quality time windows
	QuietHours []int `yaml:"quietHours"`
}

type LLMConfig struct {
	Provider string `yaml:"provider"` // "openai" or "none"
	Model    string `yaml:"model"`
	// If empty, read from env OPENAI_API_KEY
	APIKey string `yaml:"apiKey"`
}

// Default returns a sensible default configuration.
func Default() Config {
	return Config{
		Account: AccountConfig{Username: ""},
		Credentials: CredentialsConfig{BearerToken: ""},
		Interests: InterestsConfig{
			Topics:   []string{"ai", "golang", "distributed systems", "product design"},
			Keywords: []string{"golang", "LLM", "vector", "consensus", "raft", "kubernetes", "observability"},
			Weights:  map[string]float64{"golang": 1.2, "LLM": 1.0, "kubernetes": 0.9},
		},
		Filters: FiltersConfig{MinOrganicScore: 0.55, MaxBotLikelihood: 0.35, Languages: []string{"en"}},
		Engagement: EngagementConfig{MaxPerHour: 6, MaxPerDay: 40, QuietHours: []int{0, 1, 2, 3, 4, 5}},
		LLM:       LLMConfig{Provider: "none", Model: "gpt-4o-mini", APIKey: ""},
	}
}

// ResolveEnv fills in config fields from environment variables if not set.
func (c *Config) ResolveEnv() {
	if c.Credentials.BearerToken == "" {
		c.Credentials.BearerToken = os.Getenv("X_BEARER_TOKEN")
	}
	if c.LLM.APIKey == "" && c.LLM.Provider == "openai" {
		c.LLM.APIKey = os.Getenv("OPENAI_API_KEY")
	}
}

// Load reads YAML config from path.
func Load(path string) (Config, error) {
	var cfg Config
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	cfg.ResolveEnv()
	return cfg, nil
}

// Save writes YAML config to path, creating directories as needed.
func Save(path string, cfg Config) error {
	if path == "" {
		return errors.New("empty path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
