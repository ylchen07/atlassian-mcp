package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	// Site represents the legacy shared hostname; kept for backwards compatibility.
	Site       string        `mapstructure:"site"`
	Jira       ServiceConfig `mapstructure:"jira"`
	Confluence ServiceConfig `mapstructure:"confluence"`
}

// ServiceConfig describes connectivity for a single Atlassian product.
type ServiceConfig struct {
	Site               string `mapstructure:"site"`
	APIBase            string `mapstructure:"api_base"`
	ServiceCredentials `mapstructure:",squash"`
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
		if cfgDir, err := os.UserConfigDir(); err == nil && cfgDir != "" {
			defaultPath := filepath.Join(cfgDir, "atlassian-mcp")
			v.AddConfigPath(defaultPath)
		}
	}

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

	cfg.applyDefaults()

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if err := c.Atlassian.Jira.validate("atlassian.jira"); err != nil {
		return err
	}

	if err := c.Atlassian.Confluence.validate("atlassian.confluence"); err != nil {
		return err
	}

	if c.Server.LogLevel == "" {
		c.Server.LogLevel = "info"
	}

	return nil
}

func (c *Config) applyDefaults() {
	root := strings.TrimSpace(c.Atlassian.Site)
	c.Atlassian.Site = root

	c.Atlassian.Jira.Site = normalizeServiceSite(c.Atlassian.Jira.Site, root)
	c.Atlassian.Jira.APIBase = strings.TrimSpace(c.Atlassian.Jira.APIBase)
	c.Atlassian.Confluence.Site = normalizeServiceSite(c.Atlassian.Confluence.Site, root)
	c.Atlassian.Confluence.APIBase = strings.TrimSpace(c.Atlassian.Confluence.APIBase)
}

func normalizeServiceSite(serviceSite, fallback string) string {
	trimmed := strings.TrimSpace(serviceSite)
	if trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(fallback)
}

func (s ServiceConfig) validate(name string) error {
	if strings.TrimSpace(s.Site) == "" {
		return fmt.Errorf("config: %s.site is required", name)
	}
	if err := s.ServiceCredentials.validate(name); err != nil {
		return err
	}
	return nil
}

func (s ServiceCredentials) validate(name string) error {
	if s.OAuthToken == "" && (s.Email == "" || s.APIToken == "") {
		return fmt.Errorf("config: %s requires either oauth_token or email/api_token", name)
	}
	return nil
}
