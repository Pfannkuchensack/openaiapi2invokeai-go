package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	flag "github.com/spf13/pflag"
)

type Config struct {
	ListenIP  string        `toml:"listen_ip"`
	Port      int           `toml:"port"`
	InvokeURL string        `toml:"invoke_url"`
	DataDir   string        `toml:"data_dir"`
	APIKey    string        `toml:"api_key"`
	AdminUser string        `toml:"admin_user"`
	AdminPass string        `toml:"admin_pass"`
	Timeout   time.Duration `toml:"timeout"`
	NoBrowser bool          `toml:"no_browser"`
	LogLevel  string        `toml:"log_level"`
}

func DefaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".invoke-openai-proxy"
	}
	return filepath.Join(home, ".invoke-openai-proxy")
}

func Load() (*Config, error) {
	cfg := &Config{}

	// Define flags
	flag.StringVar(&cfg.ListenIP, "listen-ip", "127.0.0.1", "Bind address")
	flag.IntVar(&cfg.Port, "port", 8080, "Listen port")
	flag.StringVar(&cfg.InvokeURL, "invoke-url", "http://127.0.0.1:9090", "InvokeAI base URL")
	flag.StringVar(&cfg.DataDir, "data-dir", DefaultDataDir(), "Data directory for workflows and registry")
	flag.StringVar(&cfg.APIKey, "api-key", "", "Optional Bearer token for API auth")
	flag.StringVar(&cfg.AdminUser, "admin-user", "", "Basic-auth user for /admin")
	flag.StringVar(&cfg.AdminPass, "admin-pass", "", "Basic-auth password for /admin")
	flag.DurationVar(&cfg.Timeout, "timeout", 300*time.Second, "Max wait per generation")
	flag.BoolVar(&cfg.NoBrowser, "no-browser", false, "Don't auto-open admin on first run")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "Log level (debug/info/warn/error)")
	flag.Parse()

	// Layer 2: config file (overwritten by flags if set)
	cfgFile := filepath.Join(cfg.DataDir, "config.toml")
	if _, err := os.Stat(cfgFile); err == nil {
		var fileCfg Config
		if _, err := toml.DecodeFile(cfgFile, &fileCfg); err != nil {
			return nil, fmt.Errorf("parsing config file: %w", err)
		}
		applyFileDefaults(cfg, &fileCfg)
	}

	// Layer 3: environment variables (override file, but flags still win)
	applyEnv(cfg)

	return cfg, nil
}

func applyFileDefaults(cfg *Config, fileCfg *Config) {
	// Only apply file values for flags that weren't explicitly set
	flag.CommandLine.Visit(func(f *flag.Flag) {})

	if !flagChanged("listen-ip") && fileCfg.ListenIP != "" {
		cfg.ListenIP = fileCfg.ListenIP
	}
	if !flagChanged("port") && fileCfg.Port != 0 {
		cfg.Port = fileCfg.Port
	}
	if !flagChanged("invoke-url") && fileCfg.InvokeURL != "" {
		cfg.InvokeURL = fileCfg.InvokeURL
	}
	if !flagChanged("data-dir") && fileCfg.DataDir != "" {
		cfg.DataDir = fileCfg.DataDir
	}
	if !flagChanged("api-key") && fileCfg.APIKey != "" {
		cfg.APIKey = fileCfg.APIKey
	}
	if !flagChanged("admin-user") && fileCfg.AdminUser != "" {
		cfg.AdminUser = fileCfg.AdminUser
	}
	if !flagChanged("admin-pass") && fileCfg.AdminPass != "" {
		cfg.AdminPass = fileCfg.AdminPass
	}
	if !flagChanged("timeout") && fileCfg.Timeout != 0 {
		cfg.Timeout = fileCfg.Timeout
	}
	if !flagChanged("no-browser") && fileCfg.NoBrowser {
		cfg.NoBrowser = fileCfg.NoBrowser
	}
	if !flagChanged("log-level") && fileCfg.LogLevel != "" {
		cfg.LogLevel = fileCfg.LogLevel
	}
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("PROXY_LISTEN_IP"); v != "" && !flagChanged("listen-ip") {
		cfg.ListenIP = v
	}
	if v := os.Getenv("PROXY_PORT"); v != "" && !flagChanged("port") {
		fmt.Sscanf(v, "%d", &cfg.Port)
	}
	if v := os.Getenv("INVOKE_URL"); v != "" && !flagChanged("invoke-url") {
		cfg.InvokeURL = v
	}
	if v := os.Getenv("PROXY_DATA_DIR"); v != "" && !flagChanged("data-dir") {
		cfg.DataDir = v
	}
	if v := os.Getenv("PROXY_API_KEY"); v != "" && !flagChanged("api-key") {
		cfg.APIKey = v
	}
	if v := os.Getenv("PROXY_ADMIN_USER"); v != "" && !flagChanged("admin-user") {
		cfg.AdminUser = v
	}
	if v := os.Getenv("PROXY_ADMIN_PASS"); v != "" && !flagChanged("admin-pass") {
		cfg.AdminPass = v
	}
	if v := os.Getenv("PROXY_TIMEOUT"); v != "" && !flagChanged("timeout") {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Timeout = d
		}
	}
	if v := os.Getenv("PROXY_LOG_LEVEL"); v != "" && !flagChanged("log-level") {
		cfg.LogLevel = v
	}
}

func flagChanged(name string) bool {
	f := flag.CommandLine.Lookup(name)
	return f != nil && f.Changed
}

func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.ListenIP, c.Port)
}
