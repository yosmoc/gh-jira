# gh-jira

Create new PR based on Jira issue ID

## Install

```bash
gh extension install yosmoc/gh-jira
```

## Pre requirements

This extension uses following environmental variables. Please set these variables before using this script.

- JIRA_DOMAIN
- JIRA_API_TOKEN

## Usage

```bash
gh jira JIRA-123
```

or

```bash
echo "JIRA-123" | gh jira
```
