// Package config loads and normalizes daemon configuration.
package config

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultServiceName = "tts-onnx"
	defaultListenAddr  = "127.0.0.1:18741"
)

type Config struct {
	ServiceName  string        `yaml:"service_name"`
	ListenAddr   string        `yaml:"listen_addr"`
	BearerToken  string        `yaml:"bearer_token"`
	PlayOnSpeak  bool          `yaml:"play_on_speak"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`

	MaxConcurrentRequests int           `yaml:"max_concurrent_requests"`
	SynthTimeout          time.Duration `yaml:"synth_timeout"`
	ModelIdleUnload       time.Duration `yaml:"model_idle_unload"`
	MaxTextChars          int           `yaml:"max_text_chars"`
	MaxPayloadBytes       int64         `yaml:"max_payload_bytes"`

	ConfigPath string `yaml:"-"`
	StateDir   string `yaml:"state_dir"`
	CacheDir   string `yaml:"cache_dir"`
	DataDir    string `yaml:"data_dir"`
	LogDir     string `yaml:"log_dir"`
	TempDir    string `yaml:"temp_dir"`

	ModelManifestPath string `yaml:"model_manifest_path"`
	Engine            string `yaml:"engine"`
}

func defaultXDG(envKey, fallback string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fallback
	}
	return filepath.Join(home, fallback)
}

func Default() Config {
	serviceName := defaultServiceName
	xdgConfig := defaultXDG("XDG_CONFIG_HOME", ".config")
	xdgData := defaultXDG("XDG_DATA_HOME", ".local/share")
	xdgState := defaultXDG("XDG_STATE_HOME", ".local/state")
	xdgCache := defaultXDG("XDG_CACHE_HOME", ".cache")
	cfgDir := filepath.Join(xdgConfig, serviceName)
	dataDir := filepath.Join(xdgData, serviceName)
	stateDir := filepath.Join(xdgState, serviceName)
	cacheDir := filepath.Join(xdgCache, serviceName)

	return Config{
		ServiceName:           serviceName,
		ListenAddr:            defaultListenAddr,
		PlayOnSpeak:           false,
		ReadTimeout:           15 * time.Second,
		WriteTimeout:          60 * time.Second,
		IdleTimeout:           120 * time.Second,
		MaxConcurrentRequests: 2,
		SynthTimeout:          25 * time.Second,
		ModelIdleUnload:       10 * time.Minute,
		MaxTextChars:          1200,
		MaxPayloadBytes:       1 << 20,
		ConfigPath:            filepath.Join(cfgDir, "config.yaml"),
		StateDir:              stateDir,
		CacheDir:              cacheDir,
		DataDir:               dataDir,
		LogDir:                filepath.Join(stateDir, "logs"),
		TempDir:               filepath.Join(dataDir, "tmp"),
		ModelManifestPath:     filepath.Join(dataDir, "models", "manifest.json"),
		Engine:                "mock",
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		path = cfg.ConfigPath
	}
	cfg.ConfigPath = path

	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	applyDefaults(&cfg)
	return cfg, nil
}

func applyDefaults(cfg *Config) {
	def := Default()
	if cfg.ServiceName == "" {
		cfg.ServiceName = def.ServiceName
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = def.ListenAddr
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = def.ReadTimeout
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = def.WriteTimeout
	}
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = def.IdleTimeout
	}
	if cfg.MaxConcurrentRequests <= 0 {
		cfg.MaxConcurrentRequests = def.MaxConcurrentRequests
	}
	if cfg.SynthTimeout <= 0 {
		cfg.SynthTimeout = def.SynthTimeout
	}
	if cfg.ModelIdleUnload <= 0 {
		cfg.ModelIdleUnload = def.ModelIdleUnload
	}
	if cfg.MaxTextChars <= 0 {
		cfg.MaxTextChars = def.MaxTextChars
	}
	if cfg.MaxPayloadBytes <= 0 {
		cfg.MaxPayloadBytes = def.MaxPayloadBytes
	}
	if cfg.StateDir == "" {
		cfg.StateDir = def.StateDir
	}
	if cfg.CacheDir == "" {
		cfg.CacheDir = def.CacheDir
	}
	if cfg.DataDir == "" {
		cfg.DataDir = def.DataDir
	}
	if cfg.LogDir == "" {
		cfg.LogDir = filepath.Join(cfg.StateDir, "logs")
	}
	if cfg.TempDir == "" {
		cfg.TempDir = filepath.Join(cfg.DataDir, "tmp")
	}
	if cfg.ModelManifestPath == "" {
		cfg.ModelManifestPath = filepath.Join(cfg.DataDir, "models", "manifest.json")
	}
	if cfg.Engine == "" {
		cfg.Engine = def.Engine
	}
}

func (c Config) EnsureDirs() error {
	dirs := []string{
		c.StateDir,
		c.CacheDir,
		c.DataDir,
		c.LogDir,
		c.TempDir,
		filepath.Dir(c.ModelManifestPath),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}
