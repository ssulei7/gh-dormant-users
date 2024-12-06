# gh-dormant-users

## Overview

This is a GitHub CLI extension that helps you identify dormant users in your organization. It checks for commit, issue, and pull request activity and generates a CSV report of dormant users.

## Installation

```bash
gh extension install ssulei7/gh-dormant-users
``` 

## Usage 

```zsh

gh dormant-users report [flags]

Flags:
      --date string       The date from which to start looking for activity. Max 3 months in the past. (required)
  -e, --email             Check if user has an email 
  -h, --help              help for report
    --org-name string   The name of the organization to report upon (required)
```

## Example

```zsh
    gh dormant-users report --date "Mar 1 2024" --org-name foobar
```
## Contributing

This is a work in progress, and I am looking for contributors. Please feel free to open an issue or PR if you have any feedback or would like to contribute.