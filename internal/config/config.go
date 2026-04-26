package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	GroqAPIKey      string
	LineAccessToken string
	LineUserID      string
	Broadcast       bool
}

// Load reads env vars from .env (local dev) or OS environment (production).
// If .env is absent the error is silently ignored — GitHub Secrets injects vars directly.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		GroqAPIKey:      os.Getenv("GROQ_API_KEY"),
		LineAccessToken: os.Getenv("LINE_CHANNEL_ACCESS_TOKEN"),
		LineUserID:      os.Getenv("LINE_USER_ID"),
		Broadcast:       os.Getenv("LINE_BROADCAST") == "true",
	}

	return cfg, cfg.validate()
}

func (c *Config) validate() error {
	required := map[string]string{
		"GROQ_API_KEY":              c.GroqAPIKey,
		"LINE_CHANNEL_ACCESS_TOKEN": c.LineAccessToken,
	}
	if !c.Broadcast {
		required["LINE_USER_ID"] = c.LineUserID
	}
	for name, val := range required {
		if val == "" {
			return fmt.Errorf("missing required env var: %s", name)
		}
	}
	return nil
}
