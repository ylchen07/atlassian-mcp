package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the full application configuration loaded from file/env.
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Atlassian AtlassianConfig `mapstructure:"atlassian"`
}

// ServerConfig holds server-specific options.
type ServerConfig struct {
	LogLevel string `mapstructure:"log_level"`
}

// AtlassianConfig encapsulates Jira and Confluence settings.
type AtlassianConfig struct {
	Site       string             `mapstructure:"site"`
	Jira       ServiceCredentials `mapstructure:"jira"`
	Confluence ServiceCredentials `mapstructure:"confluence"`
}

// ServiceCredentials describes authentication for a single Atlassian product.
type ServiceCredentials struct {
	Email      string `mapstructure:"email"`
	APIToken   string `mapstructure:"api_token"`
	OAuthToken string `mapstructure:"oauth_token"`
}

// Load reads configuration from the provided directory and environment variables.
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	if path != "" {
		info, err := os.Stat(path)
		if err == nil && info.IsDir() {
			v.AddConfigPath(path)
		} else {
			v.SetConfigFile(path)
		}
	} else {
		v.AddConfigPath(".")
	}

	v.SetEnvPrefix("jira_mcp")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("server.log_level", "info")

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("config: read: %w", err)
		}
	}

	cfg := new(Config)
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("config: unmarshal: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.Atlassian.Site == "" {
		return fmt.Errorf("config: atlassian.site is required")
	}

	if err := c.Atlassian.Jira.validate("jira"); err != nil {
		return err
	}

	if err := c.Atlassian.Confluence.validate("confluence"); err != nil {
		return err
	}

	if c.Server.LogLevel == "" {
		c.Server.LogLevel = "info"
	}

	return nil
}

func (s ServiceCredentials) validate(name string) error {
	if s.OAuthToken == "" && (s.Email == "" || s.APIToken == "") {
		return fmt.Errorf("config: %s requires either oauth_token or email/api_token", name)
	}
	return nil
}
