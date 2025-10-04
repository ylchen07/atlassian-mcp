package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"log/slog"

	atlassianclient "gitlab.com/your-org/jira-mcp/internal/atlassian"
	"gitlab.com/your-org/jira-mcp/internal/config"
	"gitlab.com/your-org/jira-mcp/internal/confluence"
	"gitlab.com/your-org/jira-mcp/internal/jira"
	mcpserver "gitlab.com/your-org/jira-mcp/internal/mcp"
	"gitlab.com/your-org/jira-mcp/internal/state"
	"gitlab.com/your-org/jira-mcp/pkg/logging"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	var cfgPath string
	flag.StringVar(&cfgPath, "config", "", "Path to configuration directory or file")
	flag.Parse()

	cfg, err := config.Load(cfgPath)
	if err != nil {
		slog.Default().Error("failed to load configuration", slog.Any("error", err))
		os.Exit(1)
	}

	logger := logging.New(cfg.Server.LogLevel)

	siteBase := ensureHTTPS(cfg.Atlassian.Site)
	jiraAPI := fmt.Sprintf("%s/rest/api/3", siteBase)
	confluenceAPI := fmt.Sprintf("%s/wiki/rest/api", siteBase)

	jiraClient, err := atlassianclient.NewClient(jiraAPI, cfg.Atlassian.Jira, logger)
	if err != nil {
		logger.Error("failed to initialize Jira client", slog.Any("error", err))
		os.Exit(1)
	}

	confluenceClient, err := atlassianclient.NewClient(confluenceAPI, cfg.Atlassian.Confluence, logger)
	if err != nil {
		logger.Error("failed to initialize Confluence client", slog.Any("error", err))
		os.Exit(1)
	}

	stateCache := state.NewCache()

	jiraService := jira.NewService(jiraClient)
	confluenceService := confluence.NewService(confluenceClient)

	srv := mcpserver.NewServer(mcpserver.Dependencies{
		JiraService:       jiraService,
		ConfluenceService: confluenceService,
		Cache:             stateCache,
		JiraBaseURL:       siteBase,
		ConfluenceBaseURL: fmt.Sprintf("%s/wiki", siteBase),
		Logger:            logger,
	})

	if err := server.ServeStdio(srv); err != nil {
		logger.Error("stdio server terminated", slog.Any("error", err))
		os.Exit(1)
	}
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
