package main

import (
	"flag"
	"os"
	"strings"

	"log/slog"

	atlassianclient "gitlab.com/your-org/atlassian-mcp/internal/atlassian"
	"gitlab.com/your-org/atlassian-mcp/internal/config"
	"gitlab.com/your-org/atlassian-mcp/internal/confluence"
	"gitlab.com/your-org/atlassian-mcp/internal/jira"
	mcpserver "gitlab.com/your-org/atlassian-mcp/internal/mcp"
	"gitlab.com/your-org/atlassian-mcp/internal/state"
	"gitlab.com/your-org/atlassian-mcp/pkg/logging"

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
		os.Exit(1)
	}

	confluenceClient, err := atlassianclient.NewClient(confluenceAPI, cfg.Atlassian.Confluence.ServiceCredentials, logger)
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
		JiraBaseURL:       jiraSite,
		ConfluenceBaseURL: buildConfluenceUIBase(confluenceSite),
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
