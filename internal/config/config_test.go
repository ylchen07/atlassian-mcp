package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestServiceCredentialsValidate(t *testing.T) {
	t.Parallel()

	creds := ServiceCredentials{Email: "user@example.com", APIToken: "token"}
	if err := creds.validate("jira"); err != nil {
		t.Fatalf("expected credentials to be valid, got %v", err)
	}

	creds = ServiceCredentials{OAuthToken: "token"}
	if err := creds.validate("jira"); err != nil {
		t.Fatalf("expected oauth credentials to be valid, got %v", err)
	}

	creds = ServiceCredentials{Email: "user@example.com"}
	if err := creds.validate("jira"); err == nil {
		t.Fatalf("expected error for incomplete credentials")
	}
}

func TestServiceConfigValidateRequiresSite(t *testing.T) {
	t.Parallel()

	cfg := ServiceConfig{
		ServiceCredentials: ServiceCredentials{Email: "user@example.com", APIToken: "token"},
	}

	if err := cfg.validate("atlassian.jira"); err == nil {
		t.Fatalf("expected validation error when site missing")
	}
}

func TestConfigApplyDefaultsSiteFallback(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Atlassian: AtlassianConfig{
			Site: "example.atlassian.net",
			Jira: ServiceConfig{
				APIBase:            " https://jira.example.com/rest/api/3/ ",
				ServiceCredentials: ServiceCredentials{Email: "user@example.com", APIToken: "token"},
			},
			Confluence: ServiceConfig{
				APIBase:            " https://confluence.example.com/wiki/rest/api/ ",
				ServiceCredentials: ServiceCredentials{Email: "user@example.com", APIToken: "token"},
			},
		},
	}

	cfg.applyDefaults()

	if cfg.Atlassian.Jira.Site != "example.atlassian.net" {
		t.Fatalf("expected jira.site fallback, got %q", cfg.Atlassian.Jira.Site)
	}
	if cfg.Atlassian.Confluence.Site != "example.atlassian.net" {
		t.Fatalf("expected confluence.site fallback, got %q", cfg.Atlassian.Confluence.Site)
	}
	if cfg.Atlassian.Jira.APIBase != "https://jira.example.com/rest/api/3/" {
		t.Fatalf("expected jira.api_base trimmed, got %q", cfg.Atlassian.Jira.APIBase)
	}
	if cfg.Atlassian.Confluence.APIBase != "https://confluence.example.com/wiki/rest/api/" {
		t.Fatalf("expected confluence.api_base trimmed, got %q", cfg.Atlassian.Confluence.APIBase)
	}
}

func TestLoad_DefaultConfigDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("APPDATA", tmp) // ensure os.UserConfigDir resolves inside the temp tree on Windows

	defaultDir := filepath.Join(tmp, "atlassian-mcp")
	if err := os.MkdirAll(defaultDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	configPath := filepath.Join(defaultDir, "config.yaml")
	configYAML := []byte(`server:
  log_level: debug
atlassian:
  jira:
    site: https://jira.example.com
    email: jira@example.com
    api_token: token
  confluence:
    site: https://confluence.example.com
    email: conf@example.com
    api_token: token
`)

	if err := os.WriteFile(configPath, configYAML, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got, want := cfg.Server.LogLevel, "debug"; got != want {
		t.Fatalf("Server.LogLevel = %q, want %q", got, want)
	}

	if got, want := cfg.Atlassian.Jira.Site, "https://jira.example.com"; got != want {
		t.Fatalf("Jira.Site = %q, want %q", got, want)
	}

	if got, want := cfg.Atlassian.Confluence.Site, "https://confluence.example.com"; got != want {
		t.Fatalf("Confluence.Site = %q, want %q", got, want)
	}
}
