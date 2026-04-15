package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Cookie string `json:"cookie"`
}

// DefaultConfigPath is the default path for the config file
var DefaultConfigPath = filepath.Join(os.Getenv("HOME"), ".quark-nd-disk", "config.json")

// Load reads the config file from the given path
func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if cfg.Cookie == "" {
		return nil, fmt.Errorf("cookie is required in config file")
	}

	return &cfg, nil
}

// LoadOrEmpty reads the config file, returns empty config if not exists
func LoadOrEmpty(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// Init creates a new config file with default values
func Init(path string) error {
	if path == "" {
		path = DefaultConfigPath
	}

	// Check if config already exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config file already exists at %s", path)
	}

	// Create directory if not exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create default config
	cfg := &Config{
		Cookie: "",
	}

	return Save(path, cfg)
}

// Save writes the config to file
func Save(path string, cfg *Config) error {
	if path == "" {
		path = DefaultConfigPath
	}

	// Create directory if not exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Set sets a config value by key
func Set(path, key, value string) error {
	if path == "" {
		path = DefaultConfigPath
	}

	cfg, err := LoadOrEmpty(path)
	if err != nil {
		return err
	}

	switch key {
	case "cookie":
		cfg.Cookie = value
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	return Save(path, cfg)
}

// Get gets a config value by key
func Get(path, key string) (string, error) {
	if path == "" {
		path = DefaultConfigPath
	}

	cfg, err := LoadOrEmpty(path)
	if err != nil {
		return "", err
	}

	switch key {
	case "cookie":
		if cfg.Cookie == "" {
			return "", fmt.Errorf("cookie is not set")
		}
		// Mask the cookie for security
		if len(cfg.Cookie) > 20 {
			return cfg.Cookie[:10] + "..." + cfg.Cookie[len(cfg.Cookie)-10:], nil
		}
		return cfg.Cookie[:5] + "...", nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

// Show shows all config values
func Show(path string) (map[string]string, error) {
	if path == "" {
		path = DefaultConfigPath
	}

	cfg, err := LoadOrEmpty(path)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)

	// Mask cookie for security
	if cfg.Cookie != "" {
		if len(cfg.Cookie) > 20 {
			result["cookie"] = cfg.Cookie[:10] + "..." + cfg.Cookie[len(cfg.Cookie)-10:]
		} else {
			result["cookie"] = cfg.Cookie[:5] + "..."
		}
	} else {
		result["cookie"] = "(not set)"
	}

	return result, nil
}
