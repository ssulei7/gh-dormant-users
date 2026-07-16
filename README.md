# gh-dormant-users

## Overview

`gh-dormant-users` is a GitHub CLI extension that helps you identify dormant users in your organization. It checks for various types of activity such as commits, issues, issue comments, and pull request comments, and generates a CSV report of dormant users. This tool is useful for maintaining active participation in your organization's repositories.

## Installation

To install the extension, use the following command:

```bash
gh extension install ssulei7/gh-dormant-users
```

## Usage

This extension provides two commands: `report` and `analyze`.

### Report Command

The `report` command generates a report of dormant users based on specified criteria.

```zsh
gh dormant-users report [flags]
```

#### Flags

- `--date string`: The date from which to start looking for activity. Max 3 months in the past. (required)
- `-e, --email`: Check if user has an email.
- `--org-name string`: The name of the organization to report upon. (required)
- `--activity-types strings`: Comma-separated list of activity types to check (commits, issues, issue-comments, pr-comments). Default is all types.
- `--request-mode string`: API request mode. `bounded` uses controlled concurrency (default); `safe` sends requests serially.
- `--initial-concurrency int`: Initial concurrent requests in bounded mode (default 5).
- `--max-concurrency int`: Adaptive concurrency ceiling in bounded mode, from 1 to 15 (default 15).
- `--requests-per-second float`: Global request-rate cap from 0 to 15 (default 10).
- `--rate-limit-reserve int`: Percentage of the primary rate limit kept in reserve (default 10).
- `--cache-dir string`: Directory for the strict ETag response cache.
- `--no-cache`: Disable the persistent response cache.
- `--clear-cache`: Clear the response cache before collecting data.

### Example

To generate a report for the organization `foobar` starting from March 1, 2024, and checking all activity types:

```zsh
gh dormant-users report --date "Mar 1 2024" --org-name foobar
```

To generate a report for the organization `foobar` starting from March 1, 2024, and only checking commit and issue activity:

```zsh
gh dormant-users report --date "Mar 1 2024" --org-name foobar --activity-types commits,issues
```

## Output

The tool generates a CSV report of dormant users and displays a bar chart of active vs. inactive users. The CSV file is saved in the current directory with the name `<org-name>-dormant-users.csv`.

### API collection and rate limits

The default `bounded` request mode starts with five concurrent requests and adaptively scales toward a ceiling of 15 only when measured latency prevents the collector from reaching its 10 requests/second start-rate cap. Concurrency is halved after secondary-limit responses and increases again only after a cooldown. The request-rate cap remains 600 requests per minute, below GitHub's published 900-point-per-minute REST ceiling for standard `GET` requests. Use `--request-mode safe` to pin concurrency to one. GitHub can enforce undisclosed secondary limits in either mode.

Responses with an `ETag` or `Last-Modified` validator are cached under the operating system's user cache directory. Every later run still revalidates each cached response with GitHub; the tool never serves intentionally stale data. An authenticated `304 Not Modified` response does not consume the primary REST rate limit, but it can still contribute to secondary limits. Cache files can contain private repository and user activity data and are written with user-only permissions.

GitHub CLI OAuth requests share the authenticated user's primary allowance with other personal access tokens, OAuth apps, and GitHub Apps acting on that user's behalf. The collector runs until the configured primary reserve is reached, then waits for reset; it also honors `Retry-After` and reports request/cache statistics at the end of a run. Fresh responses still count toward the primary limit; no client can guarantee avoidance of GitHub's undisclosed secondary-limit conditions.

### CSV Schema

The generated CSV file has the following schema:

| Username | Email            | Active | ActivityTypes  |
|----------|------------------|--------|----------------|
| user1    | user1@domain.com | true   | commits,issues |
| user2    | user2@domain.com | false  | ...            |
| ...      | ...              | ...    | ...            |

- **Username**: The GitHub username of the user.
- **Email**: The email address of the user (if available).
- **Active**: A boolean value indicating whether the user is active or not.
- **ActivityTypes**: A comma-separated list of activity types (commits, issues, issue-comments, pr-comments) for each user.

---

## Analyze Command

The `analyze` command uses GitHub Copilot to provide AI-powered analysis of your dormant user CSV reports.

```zsh
gh dormant-users analyze [flags]
```

### Prerequisites

The analyze command requires the GitHub Copilot CLI, it can be found here:
https://github.com/github/copilot-cli

### Flags

- `-f, --file string`: Path to the CSV file to analyze (required)
- `-t, --template string`: Analysis template to use (default: "summary")
- `-p, --prompt string`: Custom prompt (only used with 'custom' template)
- `--list-templates`: List available analysis templates
- `--check-copilot`: Check if Copilot CLI is available
- `--prompt-only`: Generate the prompt without sending to Copilot (useful for debugging)

### Analysis Templates

| Template | Description |
|----------|-------------|
| `summary` | Executive summary with key metrics and health assessment |
| `trends` | Activity patterns and engagement recommendations |
| `risk` | Security and compliance risk assessment |
| `recommendations` | Actionable steps for user lifecycle management |
| `custom` | Custom analysis with your own prompt |

### Examples

**Generate a summary analysis:**
```zsh
gh dormant-users analyze -f myorg-dormant-users.csv -t summary
```

**Get security risk assessment:**
```zsh
gh dormant-users analyze -f myorg-dormant-users.csv -t risk
```

**Run custom analysis:**
```zsh
gh dormant-users analyze -f myorg-dormant-users.csv -t custom -p "Which teams have the highest dormancy rates?"
```

**List available templates:**
```zsh
gh dormant-users analyze --list-templates
```

**Preview the prompt without calling Copilot:**
```zsh
gh dormant-users analyze -f myorg-dormant-users.csv -t summary --prompt-only
```

For more details on how the analyzer works, see [docs/analyzer.md](docs/analyzer.md).

---

## Contributing

This is a work in progress, and contributions are welcome. Please feel free to open an issue or PR if you have any feedback or would like to contribute.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.