package client

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// configDirOverride allows tests to redirect config/stats to a temp directory.
var configDirOverride string

type Config struct {
	Passphrase   string `toml:"passphrase"`
	Server       string `toml:"server"`
	AnthropicKey string `toml:"anthropic_key"`
	DefaultPack  string `toml:"default_pack"`
	DefaultType  string `toml:"default_type"`
	ImageMode    string `toml:"image_mode"`
}

func DefaultConfig() Config {
	return Config{
		Server:      "https://tinycrawl.puddingtime.net",
		DefaultPack: "the-dark-below",
		DefaultType: "sprint",
		ImageMode:   "auto",
	}
}

func ConfigDir() string {
	if configDirOverride != "" {
		return configDirOverride
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".tinycrawl")
}

func LoadConfig() (Config, error) {
	cfg := DefaultConfig()
	path := filepath.Join(ConfigDir(), "config.toml")
	_, err := toml.DecodeFile(path, &cfg)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	return cfg, err
}

// decodeToml is a thin wrapper used by tests.
func decodeToml(path string, cfg *Config) (toml.MetaData, error) {
	return toml.DecodeFile(path, cfg)
}
