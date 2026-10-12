package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Config struct {
	Cookie string `json:"cookie"`
}

// DefaultConfigPath is the default path for the config file.
// 用 os.UserHomeDir 取用户目录：Windows 上非 shell 启动时 HOME 常为空，
// os.Getenv("HOME") 会退化成相对路径，把配置写到进程当前目录。
var DefaultConfigPath = defaultConfigPath()

func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = os.Getenv("HOME")
	}
	if home == "" {
		home = "."
	}
	return filepath.Join(home, ".quark-nd-disk", "config.json")
}

// Watcher 提供"总是最新"的 Cookie：按配置文件修改时间缓存，
// 文件没变就用缓存值。这样 login 重写配置后，正在运行的实例下次取用就能拿到新值，
// 不必重启（多个实例共用同一份配置，登录一次即可）。
type Watcher struct {
	path string

	mu     sync.Mutex
	cookie string
	mod    time.Time
	size   int64
	valid  bool
}

// NewWatcher 创建监视器；path 为空时用默认配置路径。
func NewWatcher(path string) *Watcher {
	if path == "" {
		path = DefaultConfigPath
	}
	return &Watcher{path: path}
}

// Cookie 返回当前配置里的 Cookie。
func (w *Watcher) Cookie() string {
	w.mu.Lock()
	defer w.mu.Unlock()

	st, err := os.Stat(w.path)
	if err == nil && w.valid && st.ModTime().Equal(w.mod) && st.Size() == w.size {
		return w.cookie
	}
	cfg, err := Load(w.path)
	if err != nil {
		return w.cookie // 读失败（比如正写到一半）就沿用上一次可用的值
	}
	w.cookie, w.valid = cfg.Cookie, true
	if st != nil {
		w.mod, w.size = st.ModTime(), st.Size()
	}
	return w.cookie
}

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
