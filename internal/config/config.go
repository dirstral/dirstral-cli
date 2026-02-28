package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/joho/godotenv"
)

type Config struct {
	MCP        MCPConfig        `toml:"mcp"`
	Model      string           `toml:"model"`
	Verbose    bool             `toml:"verbose"`
	ElevenLabs ElevenLabsConfig `toml:"elevenlabs"`
	Host       HostConfig       `toml:"host"`
}

type MCPConfig struct {
	URL       string `toml:"url"`
	Transport string `toml:"transport"`
}

type ElevenLabsConfig struct {
	BaseURL string `toml:"base_url"`
	Voice   string `toml:"voice"`
}

type HostConfig struct {
	Listen  string `toml:"listen"`
	MCPPath string `toml:"mcp_path"`
}

func Default() Config {
	return Config{
		MCP: MCPConfig{
			URL:       "http://127.0.0.1:8087/mcp",
			Transport: "streamable-http",
		},
		Model:   "mistral-small-latest",
		Verbose: false,
		ElevenLabs: ElevenLabsConfig{
			BaseURL: "https://api.elevenlabs.io",
			Voice:   "Rachel",
		},
		Host: HostConfig{
			Listen:  "127.0.0.1:8087",
			MCPPath: "/mcp",
		},
	}
}

func Load() (Config, error) {
	if err := loadDotEnvPrecedence(); err != nil {
		return Config{}, err
	}

	cfg := Default()
	if err := mergeUserConfig(&cfg); err != nil {
		return Config{}, err
	}
	mergeEnv(&cfg)
	return cfg, nil
}

func StateDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "dirstral")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func loadDotEnvPrecedence() error {
	for _, name := range []string{".env", ".env.local"} {
		values, err := godotenv.Read(name)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			continue
		}
		for k, v := range values {
			if _, exists := os.LookupEnv(k); !exists {
				if setErr := os.Setenv(k, v); setErr != nil {
					return setErr
				}
			}
		}
	}
	return nil
}

func mergeUserConfig(cfg *Config) error {
	base, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	path := filepath.Join(base, "dirstral", "config.toml")
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	_, err = toml.DecodeFile(path, cfg)
	return err
}

func mergeEnv(cfg *Config) {
	if v := strings.TrimSpace(os.Getenv("DIRSTRAL_MCP_URL")); v != "" {
		cfg.MCP.URL = v
	}
	if v := strings.TrimSpace(os.Getenv("DIRSTRAL_MCP_TRANSPORT")); v != "" {
		cfg.MCP.Transport = v
	}
	if v := strings.TrimSpace(os.Getenv("DIRSTRAL_MODEL")); v != "" {
		cfg.Model = v
	}
	if v := strings.TrimSpace(os.Getenv("DIRSTRAL_VERBOSE")); v != "" {
		cfg.Verbose = v == "1" || strings.EqualFold(v, "true")
	}
	if v := strings.TrimSpace(os.Getenv("ELEVENLABS_BASE_URL")); v != "" {
		cfg.ElevenLabs.BaseURL = v
	}
	if v := strings.TrimSpace(os.Getenv("DIRSTRAL_VOICE")); v != "" {
		cfg.ElevenLabs.Voice = v
	}
	if v := strings.TrimSpace(os.Getenv("DIRSTRAL_HOST_LISTEN")); v != "" {
		cfg.Host.Listen = v
	}
	if v := strings.TrimSpace(os.Getenv("DIRSTRAL_HOST_MCP_PATH")); v != "" {
		cfg.Host.MCPPath = v
	}
}
