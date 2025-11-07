package config

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// NetrcEntry represents credentials for a single machine in .netrc.
type NetrcEntry struct {
	Machine  string
	Login    string
	Password string
	Account  string
}

// parseNetrc reads and parses a .netrc file.
// Returns a map of machine -> NetrcEntry.
func parseNetrc(path string) (map[string]NetrcEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Not an error if file doesn't exist
		}
		return nil, fmt.Errorf("netrc: open: %w", err)
	}
	defer file.Close()

	entries := make(map[string]NetrcEntry)
	var current NetrcEntry
	var hasMachine bool

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if shouldSkipLine(line) {
			continue
		}

		tokens := strings.Fields(line)
		current, hasMachine = processTokens(tokens, current, hasMachine, entries)
	}

	// Save last entry
	saveEntry(&current, hasMachine, entries)

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("netrc: scan: %w", err)
	}

	return entries, nil
}

// shouldSkipLine returns true for empty lines and comments.
func shouldSkipLine(line string) bool {
	return line == "" || strings.HasPrefix(line, "#")
}

// saveEntry saves a netrc entry to the map if it's valid.
func saveEntry(entry *NetrcEntry, hasMachine bool, entries map[string]NetrcEntry) {
	if hasMachine && entry.Machine != "" {
		entries[entry.Machine] = *entry
	}
}

// processTokens processes a line of tokens and updates the current entry.
func processTokens(tokens []string, current NetrcEntry, hasMachine bool, entries map[string]NetrcEntry) (NetrcEntry, bool) {
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]

		switch token {
		case "machine":
			saveEntry(&current, hasMachine, entries)
			if i+1 < len(tokens) {
				current = NetrcEntry{Machine: tokens[i+1]}
				hasMachine = true
				i++
			}

		case "login":
			if i+1 < len(tokens) {
				current.Login = tokens[i+1]
				i++
			}

		case "password":
			if i+1 < len(tokens) {
				current.Password = tokens[i+1]
				i++
			}

		case "account":
			if i+1 < len(tokens) {
				current.Account = tokens[i+1]
				i++
			}

		case "default":
			saveEntry(&current, hasMachine, entries)
			current = NetrcEntry{Machine: "default"}
			hasMachine = true
		}
	}
	return current, hasMachine
}

// findNetrcPath locates the .netrc file.
// Checks NETRC environment variable first, then ~/.netrc.
func findNetrcPath() string {
	// Check NETRC environment variable
	if netrcPath := os.Getenv("NETRC"); netrcPath != "" {
		return netrcPath
	}

	// Default to ~/.netrc
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".netrc")
}

// loadNetrcCredentials attempts to load credentials from .netrc for a given site.
// Returns login and password if found, empty strings otherwise.
func loadNetrcCredentials(site string) (login, password string, err error) {
	netrcPath := findNetrcPath()
	if netrcPath == "" {
		return "", "", nil
	}

	entries, err := parseNetrc(netrcPath)
	if err != nil {
		return "", "", err
	}

	if len(entries) == 0 {
		return "", "", nil
	}

	// Extract hostname from site URL
	hostname := site
	if parsed, err := url.Parse(site); err == nil && parsed.Host != "" {
		hostname = parsed.Host
	}

	// Try exact match first
	if entry, ok := entries[hostname]; ok {
		return entry.Login, entry.Password, nil
	}

	// Try without port
	if host := strings.Split(hostname, ":")[0]; host != hostname {
		if entry, ok := entries[host]; ok {
			return entry.Login, entry.Password, nil
		}
	}

	// Try default entry
	if entry, ok := entries["default"]; ok {
		return entry.Login, entry.Password, nil
	}

	return "", "", nil
}

// applyNetrcDefaults fills in missing email/api_token from .netrc if available.
func (c *Config) applyNetrcDefaults() error {
	// Load Jira credentials from .netrc if not set
	if c.Atlassian.Jira.Site != "" && c.Atlassian.Jira.Email == "" && c.Atlassian.Jira.APIToken == "" && c.Atlassian.Jira.OAuthToken == "" {
		login, password, err := loadNetrcCredentials(c.Atlassian.Jira.Site)
		if err != nil {
			return fmt.Errorf("config: load jira netrc: %w", err)
		}
		if login != "" && password != "" {
			c.Atlassian.Jira.Email = login
			c.Atlassian.Jira.APIToken = password
		}
	}

	// Load Confluence credentials from .netrc if not set
	if c.Atlassian.Confluence.Site != "" && c.Atlassian.Confluence.Email == "" && c.Atlassian.Confluence.APIToken == "" && c.Atlassian.Confluence.OAuthToken == "" {
		login, password, err := loadNetrcCredentials(c.Atlassian.Confluence.Site)
		if err != nil {
			return fmt.Errorf("config: load confluence netrc: %w", err)
		}
		if login != "" && password != "" {
			c.Atlassian.Confluence.Email = login
			c.Atlassian.Confluence.APIToken = password
		}
	}

	return nil
}
