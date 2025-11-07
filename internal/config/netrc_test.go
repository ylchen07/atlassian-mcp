package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseNetrc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    map[string]NetrcEntry
	}{
		{
			name: "simple entry",
			content: `machine jira.example.com
login user@example.com
password secret123`,
			want: map[string]NetrcEntry{
				"jira.example.com": {
					Machine:  "jira.example.com",
					Login:    "user@example.com",
					Password: "secret123",
				},
			},
		},
		{
			name: "multiple entries",
			content: `machine jira.example.com
  login jira-user@example.com
  password jira-token

machine confluence.example.com
  login conf-user@example.com
  password conf-token`,
			want: map[string]NetrcEntry{
				"jira.example.com": {
					Machine:  "jira.example.com",
					Login:    "jira-user@example.com",
					Password: "jira-token",
				},
				"confluence.example.com": {
					Machine:  "confluence.example.com",
					Login:    "conf-user@example.com",
					Password: "conf-token",
				},
			},
		},
		{
			name: "with comments and empty lines",
			content: `# This is a comment
machine jira.example.com
  # Another comment
  login user@example.com
  password secret123

# More comments
machine api.example.com
  login api-user
  password api-pass`,
			want: map[string]NetrcEntry{
				"jira.example.com": {
					Machine:  "jira.example.com",
					Login:    "user@example.com",
					Password: "secret123",
				},
				"api.example.com": {
					Machine:  "api.example.com",
					Login:    "api-user",
					Password: "api-pass",
				},
			},
		},
		{
			name: "with account field",
			content: `machine jira.example.com
  login user@example.com
  password secret123
  account team1`,
			want: map[string]NetrcEntry{
				"jira.example.com": {
					Machine:  "jira.example.com",
					Login:    "user@example.com",
					Password: "secret123",
					Account:  "team1",
				},
			},
		},
		{
			name: "single line format",
			content: `machine jira.example.com login user@example.com password secret123`,
			want: map[string]NetrcEntry{
				"jira.example.com": {
					Machine:  "jira.example.com",
					Login:    "user@example.com",
					Password: "secret123",
				},
			},
		},
		{
			name: "default entry",
			content: `machine jira.example.com
  login user1@example.com
  password pass1

default
  login default-user@example.com
  password default-pass`,
			want: map[string]NetrcEntry{
				"jira.example.com": {
					Machine:  "jira.example.com",
					Login:    "user1@example.com",
					Password: "pass1",
				},
				"default": {
					Machine:  "default",
					Login:    "default-user@example.com",
					Password: "default-pass",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmp := t.TempDir()
			netrcPath := filepath.Join(tmp, ".netrc")

			if err := os.WriteFile(netrcPath, []byte(tt.content), 0600); err != nil {
				t.Fatalf("write netrc: %v", err)
			}

			got, err := parseNetrc(netrcPath)
			if err != nil {
				t.Fatalf("parseNetrc() error = %v", err)
			}

			if len(got) != len(tt.want) {
				t.Fatalf("parseNetrc() got %d entries, want %d", len(got), len(tt.want))
			}

			for machine, wantEntry := range tt.want {
				gotEntry, ok := got[machine]
				if !ok {
					t.Errorf("missing entry for machine %q", machine)
					continue
				}

				if gotEntry.Machine != wantEntry.Machine {
					t.Errorf("machine %q: got Machine=%q, want %q", machine, gotEntry.Machine, wantEntry.Machine)
				}
				if gotEntry.Login != wantEntry.Login {
					t.Errorf("machine %q: got Login=%q, want %q", machine, gotEntry.Login, wantEntry.Login)
				}
				if gotEntry.Password != wantEntry.Password {
					t.Errorf("machine %q: got Password=%q, want %q", machine, gotEntry.Password, wantEntry.Password)
				}
				if gotEntry.Account != wantEntry.Account {
					t.Errorf("machine %q: got Account=%q, want %q", machine, gotEntry.Account, wantEntry.Account)
				}
			}
		})
	}
}

func TestLoadNetrcCredentials(t *testing.T) {
	tests := []struct {
		name         string
		netrcContent string
		site         string
		wantLogin    string
		wantPassword string
	}{
		{
			name: "exact hostname match",
			netrcContent: `machine jira.example.com
  login user@example.com
  password secret123`,
			site:         "jira.example.com",
			wantLogin:    "user@example.com",
			wantPassword: "secret123",
		},
		{
			name: "match with URL scheme",
			netrcContent: `machine jira.example.com
  login user@example.com
  password secret123`,
			site:         "https://jira.example.com",
			wantLogin:    "user@example.com",
			wantPassword: "secret123",
		},
		{
			name: "match with URL path",
			netrcContent: `machine jira.example.com
  login user@example.com
  password secret123`,
			site:         "https://jira.example.com/rest/api/2",
			wantLogin:    "user@example.com",
			wantPassword: "secret123",
		},
		{
			name: "match without port",
			netrcContent: `machine jira.example.com
  login user@example.com
  password secret123`,
			site:         "jira.example.com:443",
			wantLogin:    "user@example.com",
			wantPassword: "secret123",
		},
		{
			name: "default fallback",
			netrcContent: `machine other.example.com
  login other@example.com
  password other-pass

default
  login default@example.com
  password default-pass`,
			site:         "jira.example.com",
			wantLogin:    "default@example.com",
			wantPassword: "default-pass",
		},
		{
			name: "no match",
			netrcContent: `machine other.example.com
  login other@example.com
  password other-pass`,
			site:         "jira.example.com",
			wantLogin:    "",
			wantPassword: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			netrcPath := filepath.Join(tmp, ".netrc")

			if err := os.WriteFile(netrcPath, []byte(tt.netrcContent), 0600); err != nil {
				t.Fatalf("write netrc: %v", err)
			}

			t.Setenv("NETRC", netrcPath)

			gotLogin, gotPassword, err := loadNetrcCredentials(tt.site)
			if err != nil {
				t.Fatalf("loadNetrcCredentials() error = %v", err)
			}

			if gotLogin != tt.wantLogin {
				t.Errorf("loadNetrcCredentials() login = %q, want %q", gotLogin, tt.wantLogin)
			}
			if gotPassword != tt.wantPassword {
				t.Errorf("loadNetrcCredentials() password = %q, want %q", gotPassword, tt.wantPassword)
			}
		})
	}
}

func TestConfigApplyNetrcDefaults(t *testing.T) {
	tmp := t.TempDir()
	netrcPath := filepath.Join(tmp, ".netrc")

	netrcContent := `machine jira.example.com
  login jira@example.com
  password jira-token

machine confluence.example.com
  login conf@example.com
  password conf-token`

	if err := os.WriteFile(netrcPath, []byte(netrcContent), 0600); err != nil {
		t.Fatalf("write netrc: %v", err)
	}

	t.Setenv("NETRC", netrcPath)

	tests := []struct {
		name      string
		config    *Config
		wantJira  ServiceCredentials
		wantConf  ServiceCredentials
		wantError bool
	}{
		{
			name: "load both from netrc",
			config: &Config{
				Atlassian: AtlassianConfig{
					Jira: ServiceConfig{
						Site: "https://jira.example.com",
					},
					Confluence: ServiceConfig{
						Site: "https://confluence.example.com",
					},
				},
			},
			wantJira: ServiceCredentials{
				Email:    "jira@example.com",
				APIToken: "jira-token",
			},
			wantConf: ServiceCredentials{
				Email:    "conf@example.com",
				APIToken: "conf-token",
			},
		},
		{
			name: "netrc only fills missing credentials",
			config: &Config{
				Atlassian: AtlassianConfig{
					Jira: ServiceConfig{
						Site: "https://jira.example.com",
						ServiceCredentials: ServiceCredentials{
							Email:    "explicit@example.com",
							APIToken: "explicit-token",
						},
					},
					Confluence: ServiceConfig{
						Site: "https://confluence.example.com",
					},
				},
			},
			wantJira: ServiceCredentials{
				Email:    "explicit@example.com",
				APIToken: "explicit-token",
			},
			wantConf: ServiceCredentials{
				Email:    "conf@example.com",
				APIToken: "conf-token",
			},
		},
		{
			name: "oauth takes precedence over netrc",
			config: &Config{
				Atlassian: AtlassianConfig{
					Jira: ServiceConfig{
						Site: "https://jira.example.com",
						ServiceCredentials: ServiceCredentials{
							OAuthToken: "oauth-token",
						},
					},
					Confluence: ServiceConfig{
						Site: "https://confluence.example.com",
					},
				},
			},
			wantJira: ServiceCredentials{
				OAuthToken: "oauth-token",
			},
			wantConf: ServiceCredentials{
				Email:    "conf@example.com",
				APIToken: "conf-token",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.applyNetrcDefaults()
			if (err != nil) != tt.wantError {
				t.Errorf("applyNetrcDefaults() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.config.Atlassian.Jira.Email != tt.wantJira.Email {
				t.Errorf("Jira.Email = %q, want %q", tt.config.Atlassian.Jira.Email, tt.wantJira.Email)
			}
			if tt.config.Atlassian.Jira.APIToken != tt.wantJira.APIToken {
				t.Errorf("Jira.APIToken = %q, want %q", tt.config.Atlassian.Jira.APIToken, tt.wantJira.APIToken)
			}
			if tt.config.Atlassian.Jira.OAuthToken != tt.wantJira.OAuthToken {
				t.Errorf("Jira.OAuthToken = %q, want %q", tt.config.Atlassian.Jira.OAuthToken, tt.wantJira.OAuthToken)
			}

			if tt.config.Atlassian.Confluence.Email != tt.wantConf.Email {
				t.Errorf("Confluence.Email = %q, want %q", tt.config.Atlassian.Confluence.Email, tt.wantConf.Email)
			}
			if tt.config.Atlassian.Confluence.APIToken != tt.wantConf.APIToken {
				t.Errorf("Confluence.APIToken = %q, want %q", tt.config.Atlassian.Confluence.APIToken, tt.wantConf.APIToken)
			}
			if tt.config.Atlassian.Confluence.OAuthToken != tt.wantConf.OAuthToken {
				t.Errorf("Confluence.OAuthToken = %q, want %q", tt.config.Atlassian.Confluence.OAuthToken, tt.wantConf.OAuthToken)
			}
		})
	}
}
