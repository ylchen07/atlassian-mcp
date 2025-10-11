package main

import (
	"fmt"
	"os"
	"strings"

	"log/slog"

	"github.com/ylchen07/atlassian-mcp/internal/config"
	"github.com/ylchen07/atlassian-mcp/internal/confluence"
	"github.com/ylchen07/atlassian-mcp/internal/jira"
	mcpserver "github.com/ylchen07/atlassian-mcp/internal/mcp"
	"github.com/ylchen07/atlassian-mcp/internal/state"
	"github.com/ylchen07/atlassian-mcp/pkg/logging"

	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

var (
	cfgPath string
	rootCmd = &cobra.Command{
		Use:   "atlassian-mcp",
		Short: "Run the Atlassian MCP stdio server",
		RunE: func(_ *cobra.Command, _ []string) error {
			return run(cfgPath)
		},
	}
)

func init() {
	rootCmd.Flags().StringVarP(&cfgPath, "config", "c", "", "Path to configuration directory or file")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(path string) error {
	cfg, err := config.Load(path)
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	logger := logging.New(cfg.Server.LogLevel)

	jiraSite := ensureHTTPS(cfg.Atlassian.Jira.Site)
	if apiOverride := ensureHTTPS(cfg.Atlassian.Jira.APIBase); apiOverride != "" {
		jiraSite = apiOverride
	}

	confluenceSite := ensureHTTPS(cfg.Atlassian.Confluence.Site)
	if apiOverride := ensureHTTPS(cfg.Atlassian.Confluence.APIBase); apiOverride != "" {
		confluenceSite = apiOverride
	}

	jiraClient, err := jira.NewV2Client(jiraSite, cfg.Atlassian.Jira.ServiceCredentials)
	if err != nil {
		logger.Error("failed to initialize Jira client", slog.Any("error", err))
		return fmt.Errorf("initialize jira client: %w", err)
	}
	jiraSite = strings.TrimRight(jiraClient.Site.String(), "/")

	confluenceClient, err := confluence.NewClient(confluenceSite, cfg.Atlassian.Confluence.ServiceCredentials)
	if err != nil {
		logger.Error("failed to initialize Confluence client", slog.Any("error", err))
		return fmt.Errorf("initialize confluence client: %w", err)
	}
	confluenceSite = strings.TrimRight(confluenceClient.Site.String(), "/")

	stateCache := state.NewCache()

	jiraService := jira.NewService(jiraClient)
	confluenceService := confluence.NewService(confluenceClient)

	srv := mcpserver.NewServer(mcpserver.Dependencies{
		JiraService:       jiraService,
		ConfluenceService: confluenceService,
		Cache:             stateCache,
		JiraBaseURL:       jiraSite,
		ConfluenceBaseURL: buildConfluenceUIBase(confluenceSite),
		Logger:            logger,
	})

	if err := server.ServeStdio(srv); err != nil {
		logger.Error("stdio server terminated", slog.Any("error", err))
		return err
	}

	return nil
}

func ensureHTTPS(site string) string {
	trimmed := strings.TrimSpace(site)
	if trimmed == "" {
		return ""
	}

	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return strings.TrimRight(trimmed, "/")
	}

	return "https://" + strings.TrimRight(trimmed, "/")
}

func buildConfluenceUIBase(site string) string {
	trimmed := strings.TrimRight(site, "/")
	if trimmed == "" {
		return ""
	}
	if strings.HasSuffix(trimmed, "/wiki") {
		return trimmed
	}
	return trimmed + "/wiki"
}
