package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
}

type ServerConfig struct {
	Addr string `yaml:"addr"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

func Load(path string) (Config, error) {
	cfg := Config{
		Server: ServerConfig{
			Addr: ":8080",
		},
		Database: DatabaseConfig{
			Path: "data/biligo.db",
		},
	}

	if path == "" {
		path = os.Getenv("BILIGO_CONFIG")
	}
	if path == "" {
		path = "config.yaml"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return Config{}, err
		}
	} else if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	if addr := os.Getenv("BILIGO_ADDR"); addr != "" {
		cfg.Server.Addr = addr
	}
	if dbPath := os.Getenv("BILIGO_DB"); dbPath != "" {
		cfg.Database.Path = dbPath
	}

	if cfg.Server.Addr == "" {
		cfg.Server.Addr = ":8080"
	}
	if cfg.Database.Path == "" {
		cfg.Database.Path = "data/biligo.db"
	}

	return cfg, nil
}
