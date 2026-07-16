package cmd

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/api"
	"github.com/spf13/cobra"
	"github.com/ssulei7/gh-dormant-users/internal/activity"
	dateUtil "github.com/ssulei7/gh-dormant-users/internal/date"
	"github.com/ssulei7/gh-dormant-users/internal/githubapi"
	"github.com/ssulei7/gh-dormant-users/internal/repository"
	"github.com/ssulei7/gh-dormant-users/internal/ui"
	"github.com/ssulei7/gh-dormant-users/internal/users"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate a report",
	RunE:  generateDormantUserReport,
}

func generateDormantUserReport(cmd *cobra.Command, args []string) error {
	// First, get all users in an organization using the gh module
	orgName, _ := cmd.Flags().GetString("org-name")
	email, _ := cmd.Flags().GetBool("email")
	date, _ := cmd.Flags().GetString("date")
	requestMode, _ := cmd.Flags().GetString("request-mode")
	initialConcurrency, _ := cmd.Flags().GetInt("initial-concurrency")
	maxConcurrency, _ := cmd.Flags().GetInt("max-concurrency")
	requestsPerSecond, _ := cmd.Flags().GetFloat64("requests-per-second")
	rateLimitReserve, _ := cmd.Flags().GetInt("rate-limit-reserve")
	cacheDir, _ := cmd.Flags().GetString("cache-dir")
	noCache, _ := cmd.Flags().GetBool("no-cache")
	clearCache, _ := cmd.Flags().GetBool("clear-cache")

	if cacheDir == "" {
		var err error
		cacheDir, err = githubapi.DefaultCacheDir()
		if err != nil {
			return err
		}
	}
	if clearCache {
		if err := githubapi.ClearCache(cacheDir); err != nil {
			return err
		}
		ui.Info("Cleared API cache at %s", cacheDir)
	}
	switch strings.ToLower(requestMode) {
	case "safe":
		initialConcurrency = 1
		maxConcurrency = 1
	case "bounded":
	default:
		return fmt.Errorf("invalid request mode %q; expected safe or bounded", requestMode)
	}

	coordinator, err := githubapi.NewCoordinator(githubapi.Config{
		Transport:          http.DefaultTransport,
		CacheDir:           cacheDir,
		CacheEnabled:       !noCache,
		InitialConcurrency: initialConcurrency,
		MaxConcurrency:     maxConcurrency,
		RequestsPerSecond:  requestsPerSecond,
		RateLimitReserve:   float64(rateLimitReserve) / 100,
	})
	if err != nil {
		return fmt.Errorf("configure GitHub API requests: %w", err)
	}
	clientOptions := func() *api.ClientOptions {
		return &api.ClientOptions{
			Transport: coordinator,
			Headers: map[string]string{
				"Accept":               "application/vnd.github+json",
				"X-GitHub-Api-Version": "2026-03-10",
			},
		}
	}
	restClient, err := gh.RESTClient(clientOptions())
	if err != nil {
		return fmt.Errorf("create REST client: %w", err)
	}
	gqlClient, err := gh.GQLClient(clientOptions())
	if err != nil {
		return fmt.Errorf("create GraphQL client: %w", err)
	}

	// Validate date is no longer than 3 months
	if err := dateUtil.ValidateDate(date); err != nil {
		return err
	}

	// Convert date to iso 8601 format
	isoDate, err := dateUtil.GetISODate(date)
	if err != nil {
		return err
	}

	users, err := users.GetOrganizationUsers(orgName, email, restClient, gqlClient)
	if err != nil {
		return err
	}

	repositories, err := repository.GetOrgRepositories(orgName, restClient)
	if err != nil {
		return err
	}

	activityTypes, _ := cmd.Flags().GetStringSlice("activity-types")

	// Now, check for activity in the organization's repositories
	ui.BoxWithTitle("Organization Info", fmt.Sprintf("Number of users: %v\nNumber of repositories: %v", len(users), len(repositories)))
	ui.Info("Checking for activity...")

	checker := activity.NewActivityChecker(maxConcurrency)
	if err := checker.CheckActivity(users, orgName, repositories, isoDate, restClient, activityTypes); err != nil {
		return fmt.Errorf("collect activity: %w", err)
	}
	checker.GenerateBarChart()

	if err := activity.GenerateUserReportCSV(users, orgName+"-dormant-users.csv"); err != nil {
		return fmt.Errorf("generate report: %w", err)
	}

	stats := coordinator.Stats()
	ui.Info(
		"API summary: %d requests, %d revalidated cache hits, %d cache errors, %d retries, %v waiting, %d/%d primary requests remaining",
		stats.Requests,
		stats.CacheHits,
		stats.CacheErrors,
		stats.Retries,
		stats.WaitDuration.Round(time.Second),
		stats.RateRemaining,
		stats.RateLimit,
	)
	endpoints := make([]string, 0, len(stats.EndpointCounts))
	for endpoint := range stats.EndpointCounts {
		endpoints = append(endpoints, endpoint)
	}
	sort.Strings(endpoints)
	counts := make([]string, 0, len(endpoints))
	for _, endpoint := range endpoints {
		counts = append(counts, fmt.Sprintf("%s=%d", endpoint, stats.EndpointCounts[endpoint]))
	}
	ui.Info("API requests by endpoint: %s", strings.Join(counts, ", "))
	ui.Info(
		"API concurrency: %d initial, %d final, %d peak, %d reductions; latency p50=%v p95=%v, achieved %.1f req/s",
		stats.InitialConcurrency,
		stats.CurrentConcurrency,
		stats.PeakConcurrency,
		stats.ConcurrencyReductions,
		stats.P50Latency.Round(time.Millisecond),
		stats.P95Latency.Round(time.Millisecond),
		stats.AchievedRequestsPerSecond,
	)
	return nil
}
