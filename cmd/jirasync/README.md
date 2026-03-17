# jirasync

**jirasync** is a centralized service that monitors GitHub pull requests across multiple repositories and automatically transitions linked Jira tickets based on PR state.

## Overview

**jirasync** runs centrally (e.g., via GitHub Actions or cron) to keep ticket statuses synchronized with PR states across multiple repositories. It dynamically fetches available Jira transitions using the Jira API, supports multi-step workflow transitions, and can send Slack notifications for visibility.

## Features

- **Cross-repository support**: Monitor PRs across multiple GitHub repositories
- **Dynamic transition detection**: Uses Jira API to fetch available transitions (no hardcoding)
- **Multi-step transitions**: Automatically navigates intermediate workflow states
- **Global batching**: Handles tickets with PRs across multiple repositories
- **Slack integration**: Optional notifications for transitioned tickets
- **Flexible date filtering**: Process PRs from specific dates or time ranges
- **Terminal state protection**: Never auto-closes tickets (On QA is the furthest state)

## PR State → Jira Transition Mapping

| PR State | Jira Target Status |
|----------|-------------------|
| Draft / Changes Requested | In Progress |
| Ready / Review Requested | Code Review |
| Approved | Code Review |
| Merged | On QA |
| Closed (without merge) | On QA |

**Note:** Tickets are never automatically closed. "On QA" is the furthest automated state. Closing tickets should be done manually after QA verification.

## Installation

```bash
# From the repository root
go install ./cmd/jirasync
```

## Testing

Before running jirasync in production, test the flow with the test harness:

```bash
# Set environment variables
export GITHUB_TOKEN=ghp_xxxxx
export JIRA_URL=https://your-jira.com
export JIRA_EMAIL=user@company.com
export JIRA_TOKEN=xxxxx
export JIRA_PR_FIELD=customfield_12345
export SLACK_WEBHOOK_URL=https://hooks.slack.com/services/xxx  # Optional

# Run test (does NOT execute transitions)
go run ./test/jirasync/main.go

# Or test with your own repo
go run ./test/jirasync/main.go myorg myrepo
```

The test harness shows:
- ✅ PRs fetched from GitHub
- ✅ PR states detected
- ✅ Jira tickets found via JQL
- ✅ Available transitions
- ✅ Which transitions would be executed
- ✅ Slack notification preview (if webhook configured)
- ❌ **Does NOT execute transitions** (safe to run)

See [test/jirasync/README.md](../../test/jirasync/README.md) for details.

## Configuration

### 1. Create Repository List

Create a YAML file with repositories to monitor:

```yaml
# repos.yaml
repositories:
  - owner: your-org
    name: backend
  - owner: your-org
    name: frontend
  - owner: your-org
    name: infrastructure
```

### 2. Set Environment Variables (Optional)

Instead of passing flags, you can set environment variables:

```bash
export GITHUB_TOKEN=ghp_xxxxx
export JIRA_URL=https://jira.yourcompany.com
export JIRA_EMAIL=yourname@company.com
export JIRA_TOKEN=xxxxx
export JIRA_PR_FIELD=customfield_12345
export SLACK_WEBHOOK_URL=https://hooks.slack.com/services/xxx  # Optional
```

## Usage

### Basic Usage

Process PRs from yesterday (default):
```bash
jirasync \
  --config repos.yaml \
  --github-token $GITHUB_TOKEN \
  --jira-url $JIRA_URL \
  --jira-email $JIRA_EMAIL \
  --jira-token $JIRA_TOKEN \
  --jira-pr-field customfield_12345
```

Process PRs since a specific date:
```bash
jirasync \
  --config repos.yaml \
  --since 2026-03-16 \
  --github-token $GITHUB_TOKEN \
  --jira-url $JIRA_URL \
  --jira-email $JIRA_EMAIL \
  --jira-token $JIRA_TOKEN \
  --jira-pr-field customfield_12345
```

With Slack notifications:
```bash
jirasync \
  --config repos.yaml \
  --github-token $GITHUB_TOKEN \
  --jira-url $JIRA_URL \
  --jira-email $JIRA_EMAIL \
  --jira-token $JIRA_TOKEN \
  --jira-pr-field customfield_12345 \
  --slack-webhook $SLACK_WEBHOOK_URL
```

Supported date formats for `--since`:
- ISO format: `2026-03-16` (YYYY-MM-DD)
- European format: `16/03/2026` (DD/MM/YYYY)
- European with dashes: `16-03-2026` (DD-MM-YYYY)

**Note:** If `--since` is not provided, jirasync defaults to processing PRs updated since yesterday (last 24 hours).

### Run as GitHub Actions Workflow

Create `.github/workflows/jirasync.yml`:

```yaml
name: Jira Sync

on:
  schedule:
    # Run daily at 9 AM UTC
    - cron: '0 9 * * *'
  workflow_dispatch:  # Allow manual triggers

jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Install jirasync
        run: go install ./cmd/jirasync

      - name: Run jirasync
        env:
          GITHUB_TOKEN: ${{ secrets.GH_PAT }}
          JIRA_URL: ${{ secrets.JIRA_URL }}
          JIRA_EMAIL: ${{ secrets.JIRA_EMAIL }}
          JIRA_TOKEN: ${{ secrets.JIRA_TOKEN }}
          JIRA_PR_FIELD: ${{ secrets.JIRA_PR_FIELD }}
          SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK_URL }}
        run: |
          # Defaults to yesterday's date automatically
          jirasync \
            --config jirasync-repos.yaml \
            --github-token "$GITHUB_TOKEN" \
            --jira-url "$JIRA_URL" \
            --jira-email "$JIRA_EMAIL" \
            --jira-token "$JIRA_TOKEN" \
            --jira-pr-field "$JIRA_PR_FIELD" \
            --slack-webhook "$SLACK_WEBHOOK_URL"
```

### Run via Cron

Add to crontab to run daily (defaults to yesterday's PRs):

```bash
0 9 * * * /path/to/jirasync --config /path/to/repos.yaml --github-token $GITHUB_TOKEN --jira-url $JIRA_URL --jira-email $JIRA_EMAIL --jira-token $JIRA_TOKEN --jira-pr-field customfield_12345 --slack-webhook $SLACK_WEBHOOK_URL >> /var/log/jirasync.log 2>&1
```

## Slack Notifications

When configured with `--slack-webhook`, jirasync sends a notification after each sync run showing:

- All tickets that were transitioned
- Current and target status for each ticket
- All PRs linked to each ticket (with repo context)
- Summary statistics (tickets in correct status, unlinked PRs)

Example Slack message:
```
The following Jira tickets were processed for workflow transitions.

SRVKP-11096 `To Do` → `On QA` ✅ APPLIED
  • tektoncd/results#1257: Change all occurences of GCS buckets...
  • tektoncd/cli#2768: Change all occurences of GCS buckets...

SRVKP-11090 `To Do` → `Code Review` ✅ APPLIED
  • tektoncd/plumbing#3223: Replace occurences of GCS buckets...

🤖 jirasync | 2 tickets processed • 5 already in correct status • 12 unlinked PRs
```

## How It Works

1. **Global Collection**: For each repository in the config, jirasync collects all PRs updated since the specified date
2. **State Detection**: Determines PR state (draft, approved, merged, closed) using GitHub API
3. **Ticket Lookup**: Finds Jira tickets linked to each PR via JQL search on the PR URL field
4. **Global Batching**: Groups all PRs by ticket across all repositories to avoid duplicate transitions
5. **Transition Strategy**: For tickets with multiple PRs, uses the "most behind" PR state to determine target status
6. **Transition Execution**:
   - Fetches available transitions from Jira API
   - Attempts direct transition first
   - Falls back to multi-step transition (max 3 steps) if needed
   - Skips tickets already in terminal states (Closed/Done)
7. **Notification**: Sends Slack notification with results (if configured)

## Multi-Step Transitions

If a direct transition isn't available, jirasync will attempt to navigate through intermediate states automatically:

```
Example: To Do → On QA (no direct path)
Step 1: To Do → In Progress
Step 2: In Progress → Code Review
Step 3: Code Review → On QA
```

The algorithm prefers common workflow states like "In Progress" and "Code Review" as intermediate steps.

## Cross-Repository Support

When a Jira ticket has PRs across multiple repositories:

1. All PRs are collected first (Phase 1)
2. Ticket is transitioned only once using the "most behind" PR state (Phase 2)
3. Slack notification shows all related PRs with repository context

Example:
```
PROJ-123 has PRs in 3 repos:
  - myorg/backend PR #42 (merged)
  - myorg/frontend PR #15 (approved)
  - myorg/api PR #7 (draft)

→ Ticket transitions to "In Progress" (uses draft state from api repo)
```

## Finding Your Jira PR Field ID

1. Open any Jira ticket in your browser
2. Append `?expand=fields` to the URL
3. Search for your PR field (usually named "Git PR URL" or similar)
4. The field ID will be something like `customfield_12345`

Alternatively, use the Jira API:
```bash
curl -u email@example.com:$JIRA_TOKEN \
  https://your-jira.com/rest/api/3/field | jq '.[] | select(.name | contains("PR"))'
```

## Logging

All operations are logged to stdout for easy integration with GitHub Actions, cron logs, or any logging system.

Example output:
```
Loaded 2 repositories from config
Processing PRs updated since 2026-03-16 (yesterday - default)

Starting jirasync for 2 repositories

=== Collecting PRs from myorg/backend ===
  Found 5 PRs updated since 2026-03-16

=== Collecting PRs from myorg/frontend ===
  Found 3 PRs updated since 2026-03-16

=== Processing Tickets (Global Batching) ===
Found 4 unique tickets across all repositories

  ✓ Transitioned PROJ-123: 'To Do' → 'Code Review' (PR state: approved)
  ✓ Transitioned PROJ-456: 'In Progress' → 'On QA' in 2 steps (PR state: merged)
    Path: 'Code Review' → 'On QA'
  ⊗ PROJ-789: Already in terminal state 'Closed' - skipping

Jirasync completed - 2 tickets transitioned

=== Sync Summary ===

myorg/backend:
  PRs processed: 5
  Tickets transitioned: 1

myorg/frontend:
  PRs processed: 3
  Tickets transitioned: 1

Total:
  PRs: 8
  Tickets transitioned: 2
  Tickets already in correct status: 3
  Unlinked PRs: 2
  Errors: 0

📢 Sending Slack notification for 2 tickets...
  ✓ Slack notification sent successfully
```

## Troubleshooting

### No tickets transitioned

- Verify PRs are linked in Jira (check the PR URL field)
- Check if tickets are already in the target status
- Ensure your Jira workflow has matching transition names
- Check the stdout/logs for error messages

### JQL search fails

- Verify the field ID is correct with the method above
- Check field permissions in Jira
- Ensure the field contains the full PR URL (not just PR number)
- Test the JQL query in Jira's issue navigator

### Transition fails

- Check if the transition exists in your workflow
- Verify user has permission to transition tickets
- Review Jira workflow configuration
- Check if required fields are missing (jirasync doesn't set custom fields during transition)

### Empty transitions array

- This usually indicates API token permissions issues
- Regenerate your Jira API token with full permissions
- Ensure the token has "Browse Projects" and "Transition Issues" permissions

### Slack notification not sent

- Verify webhook URL is correct
- Check Slack app permissions
- Test webhook manually with curl
- Check logs for specific error messages
