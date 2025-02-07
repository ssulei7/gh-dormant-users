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

## Contributing

This is a work in progress, and contributions are welcome. Please feel free to open an issue or PR if you have any feedback or would like to contribute.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.