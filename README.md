# gh-dormant-users

## Overview

`gh-dormant-users` is a GitHub CLI extension that helps you identify dormant users in your organization. It checks for various types of activity such as commits, issues, issue comments, and pull request comments, and generates a CSV report of dormant users. This tool is useful for maintaining active participation in your organization's repositories.

## Installation

To install the extension, use the following command:

```bash
gh extension install ssulei7/gh-dormant-users
```

## Usage

The primary command for this extension is `report`, which generates a report of dormant users based on specified criteria.

```zsh
gh dormant-users report [flags]
```

### Flags

- `--date string`: The date from which to start looking for activity. Max 3 months in the past. (required)
- `-e, --email`: Check if user has an email.
- `--org-name string`: The name of the organization to report upon. (required)
- `--activity-types strings`: Comma-separated list of activity types to check (commits, issues, issue-comments, pr-comments). Default is all types.

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