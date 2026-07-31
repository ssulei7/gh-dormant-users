package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func newReportTestCommand() *cobra.Command {
	command := &cobra.Command{}
	flags := command.Flags()
	flags.String("org-name", "", "")
	flags.Bool("email", false, "")
	flags.String("date", "", "")
	flags.String("request-mode", "bounded", "")
	flags.Int("initial-concurrency", 5, "")
	flags.Int("max-concurrency", 15, "")
	flags.Float64("requests-per-second", 10, "")
	flags.Int("rate-limit-reserve", 10, "")
	flags.String("cache-dir", "", "")
	flags.Bool("no-cache", false, "")
	flags.Bool("clear-cache", false, "")
	flags.StringSlice("activity-types", nil, "")
	return command
}

func setReportTestFlags(t *testing.T, command *cobra.Command, values map[string]string) {
	t.Helper()
	for name, value := range values {
		if err := command.Flags().Set(name, value); err != nil {
			t.Fatalf("set %s: %v", name, err)
		}
	}
}

func configureReportDependencies(t *testing.T, cacheDir func() (string, error), clear func(string) error) {
	t.Helper()
	oldDefault := defaultCacheDir
	oldClear := clearAPICache
	defaultCacheDir = cacheDir
	clearAPICache = clear
	t.Cleanup(func() {
		defaultCacheDir = oldDefault
		clearAPICache = oldClear
	})
}

func TestReadReportOptions(t *testing.T) {
	command := newReportTestCommand()
	setReportTestFlags(t, command, map[string]string{
		"org-name":            "example",
		"email":               "true",
		"date":                "Jul 1 2026",
		"request-mode":        "safe",
		"initial-concurrency": "4",
		"max-concurrency":     "8",
		"requests-per-second": "7.5",
		"rate-limit-reserve":  "20",
		"cache-dir":           "/cache",
		"no-cache":            "true",
		"clear-cache":         "true",
	})

	got := readReportOptions(command)
	if got.orgName != "example" || !got.email || got.date != "Jul 1 2026" || got.requestMode != "safe" {
		t.Fatalf("basic options = %#v", got)
	}
	if got.initialConcurrency != 4 || got.maxConcurrency != 8 || got.requestsPerSecond != 7.5 {
		t.Fatalf("request options = %#v", got)
	}
	if got.rateLimitReserve != 20 || got.cacheDir != "/cache" || !got.noCache || !got.clearCache {
		t.Fatalf("cache options = %#v", got)
	}
}

func TestPrepareReportOptionsSafeMode(t *testing.T) {
	cleared := ""
	configureReportDependencies(
		t,
		func() (string, error) { return "/default-cache", nil },
		func(path string) error {
			cleared = path
			return nil
		},
	)

	got, err := prepareReportOptions(reportOptions{
		requestMode:        "SAFE",
		initialConcurrency: 5,
		maxConcurrency:     15,
		clearCache:         true,
	})
	if err != nil {
		t.Fatalf("prepareReportOptions returned error: %v", err)
	}
	if got.cacheDir != "/default-cache" || cleared != "/default-cache" {
		t.Fatalf("cache dir = %q, cleared = %q", got.cacheDir, cleared)
	}
	if got.initialConcurrency != 1 || got.maxConcurrency != 1 {
		t.Fatalf("safe concurrency = %d/%d", got.initialConcurrency, got.maxConcurrency)
	}
}

func TestPrepareReportOptionsBoundedMode(t *testing.T) {
	configureReportDependencies(t, func() (string, error) {
		t.Fatal("default cache lookup should not run")
		return "", nil
	}, func(string) error {
		t.Fatal("cache clear should not run")
		return nil
	})

	got, err := prepareReportOptions(reportOptions{
		requestMode:        "bounded",
		initialConcurrency: 4,
		maxConcurrency:     8,
		cacheDir:           "/cache",
	})
	if err != nil {
		t.Fatalf("prepareReportOptions returned error: %v", err)
	}
	if got.initialConcurrency != 4 || got.maxConcurrency != 8 {
		t.Fatalf("bounded concurrency = %d/%d", got.initialConcurrency, got.maxConcurrency)
	}
}

func TestPrepareReportOptionsErrors(t *testing.T) {
	t.Run("default cache", func(t *testing.T) {
		configureReportDependencies(t, func() (string, error) {
			return "", errors.New("cache lookup failed")
		}, func(string) error { return nil })
		_, err := prepareReportOptions(reportOptions{requestMode: "bounded"})
		if err == nil || !strings.Contains(err.Error(), "cache lookup failed") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("clear cache", func(t *testing.T) {
		configureReportDependencies(t, func() (string, error) {
			return "/cache", nil
		}, func(string) error {
			return errors.New("clear failed")
		})
		_, err := prepareReportOptions(reportOptions{requestMode: "bounded", clearCache: true})
		if err == nil || !strings.Contains(err.Error(), "clear failed") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("request mode", func(t *testing.T) {
		configureReportDependencies(t, func() (string, error) {
			return "/cache", nil
		}, func(string) error { return nil })
		_, err := prepareReportOptions(reportOptions{requestMode: "turbo"})
		if err == nil || !strings.Contains(err.Error(), "invalid request mode") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestGenerateDormantUserReportRejectsInvalidMode(t *testing.T) {
	configureReportDependencies(t, func() (string, error) {
		return "/cache", nil
	}, func(string) error { return nil })
	command := newReportTestCommand()
	setReportTestFlags(t, command, map[string]string{"request-mode": "turbo"})

	err := generateDormantUserReport(command, nil)
	if err == nil || !strings.Contains(err.Error(), "invalid request mode") {
		t.Fatalf("error = %v", err)
	}
}
