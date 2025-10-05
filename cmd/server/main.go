package main

import (
	"fmt"
	"os"
	"strings"

	"log/slog"

	atlassianclient "github.com/ylchen07/atlassian-mcp/internal/atlassian"
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
	jiraAPI := buildJiraAPIBase(jiraSite)
	if apiOverride := ensureHTTPS(cfg.Atlassian.Jira.APIBase); apiOverride != "" {
		jiraAPI = strings.TrimRight(apiOverride, "/")
	}

	confluenceSite := ensureHTTPS(cfg.Atlassian.Confluence.Site)
	confluenceAPI := buildConfluenceAPIBase(confluenceSite)
	if apiOverride := ensureHTTPS(cfg.Atlassian.Confluence.APIBase); apiOverride != "" {
		confluenceAPI = strings.TrimRight(apiOverride, "/")
	}

	jiraClient, err := atlassianclient.NewClient(jiraAPI, cfg.Atlassian.Jira.ServiceCredentials, logger)
	if err != nil {
		logger.Error("failed to initialize Jira client", slog.Any("error", err))
		return fmt.Errorf("initialize jira client: %w", err)
	}

	confluenceClient, err := atlassianclient.NewClient(confluenceAPI, cfg.Atlassian.Confluence.ServiceCredentials, logger)
	if err != nil {
		logger.Error("failed to initialize Confluence client", slog.Any("error", err))
		return fmt.Errorf("initialize confluence client: %w", err)
	}

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

func buildJiraAPIBase(site string) string {
	trimmed := strings.TrimRight(site, "/")
	if trimmed == "" {
		return ""
	}
	if strings.HasSuffix(trimmed, "/rest/api/3") {
		return trimmed
	}
	return trimmed + "/rest/api/3"
}

func buildConfluenceAPIBase(site string) string {
	trimmed := strings.TrimRight(site, "/")
	if trimmed == "" {
		return ""
	}
	if strings.Contains(trimmed, "/rest/") {
		return trimmed
	}
	if strings.HasSuffix(trimmed, "/wiki") {
		return trimmed + "/rest/api"
	}
	return trimmed + "/wiki/rest/api"
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
