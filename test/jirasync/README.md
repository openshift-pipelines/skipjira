# JiraSync Test Harness

This test program allows you to test the complete jirasync flow without actually executing transitions. It validates the entire pipeline including multi-repository support, global batching, and Slack notifications.

## What It Tests

1. **GitHub PR Fetching** - Lists PRs from one or more repositories with server-side filtering
2. **PR State Detection** - Determines PR states (draft, approved, merged, closed, changes_requested)
3. **Jira JQL Search** - Finds tickets linked to PRs via custom field search
4. **Global Batching** - Collects all PRs across repositories before processing tickets
5. **Cross-Repo Handling** - Shows how tickets with PRs in multiple repos are handled
6. **Transition Discovery** - Fetches available transitions from Jira API
7. **Multi-Step Transitions** - Tests path finding through intermediate workflow states
8. **Slack Integration** - Previews notifications without sending (or sends if webhook configured)

## Usage

### Set Environment Variables

```bash
export GITHUB_TOKEN=ghp_xxxxx
export JIRA_URL=https://your-jira.com
export JIRA_EMAIL=user@company.com
export JIRA_TOKEN=xxxxx
export JIRA_PR_FIELD=customfield_12345
export SLACK_WEBHOOK_URL=https://hooks.slack.com/services/xxx  # Optional
```

**Note:** Tokens are read from environment variables but currently hardcoded in the test script for convenience. Update `test/jirasync/main.go` to use environment variables in production.

### Run the Test

```bash
# Test with default repo (tektoncd/results)
go run ./test/jirasync/main.go

# Test with a single custom repo
go run ./test/jirasync/main.go myorg myrepo

# Test with multiple repos using config file
go run ./test/jirasync/main.go --config test-repos.yaml
```

### Multi-Repository Testing

Create a config file:

```yaml
# test-repos.yaml
repositories:
  - owner: tektoncd
    name: results
  - owner: tektoncd
    name: cli
  - owner: tektoncd
    name: plumbing
```

Then run:
```bash
go run ./test/jirasync/main.go --config test-repos.yaml
```

## Example Output

```
Testing jirasync flow for 3 repository(ies)
Fetching PRs updated since 2026-03-10

=== Step 1: Connecting to Jira ===
✓ Jira client created successfully

=== Step 2: Collecting PRs from All Repositories ===

Repository: tektoncd/results
  Found 12 PRs

Repository: tektoncd/cli
  Found 8 PRs

Repository: tektoncd/plumbing
  Found 5 PRs

=== Collection Summary ===
Total repositories: 3
Total PRs collected: 25
Unique Jira tickets: 8
Tickets with PRs across multiple repos: 2

=== Step 3: Processing Tickets ===

[SRVKP-11096] Change all occurences of GCS buckets with OCI buck...
  Current Status: To Do
  Linked PRs: 2
    [tektoncd/results] PR #1257 (merged) - Change all occurences of GCS buckets with OCI buckets
  → [tektoncd/cli] PR #2768 (merged) - Change all occurences of GCS buckets with OCI buckets
  ⚠ PRs span 2 repositories
  Using most behind state: merged (from tektoncd/results PR #1257)
  Target Jira Status: Dev Complete
  Available transitions (4): Start → In Progress, Review → Code Review, QA → Dev Complete, Close → Closed
  ⚠ No direct transition to 'Dev Complete'
  → Multi-step possible: 'To Do' → 'In Progress' → 'Dev Complete'
  ✓ Would attempt multi-step transition (max 3 steps)

[SRVKP-11090] Replace occurences of GCS buckets to OCI buckets i...
  Current Status: To Do
  Linked PRs: 2
    [tektoncd/plumbing] PR #3223 (approved) - Replace occurences of GCS buckets to OCI buckets in 2 files
  → [tektoncd/results] PR #1257 (merged) - Change all occurences of GCS buckets with OCI buckets
  ⚠ PRs span 2 repositories
  Using most behind state: approved (from tektoncd/plumbing PR #3223)
  Target Jira Status: Code Review
  Available transitions (4): Start → In Progress, Review → Code Review, QA → Dev Complete, Close → Closed
  ✓ Would execute direct transition: 'To Do' → 'Code Review'
  → Using transition: 'Review' (ID: 21)

[SRVKP-10943] feat(validation): add GitHub URL validation
  Current Status: Closed
  ⊗ Ticket is in terminal state - skipping

═══════════════════════════════════════════════════════════
Overall Summary
═══════════════════════════════════════════════════════════
Total repositories: 3
Total PRs collected: 25
Unique Jira tickets: 8
Tickets with cross-repo PRs: 2
Transitions that would be executed: 5

📢 Sending Slack notification for 5 tickets...
  ✓ Slack notification sent successfully

Test complete! No actual transitions were executed.
To execute transitions, use: jirasync --config repos.yaml ...

Tip: Set SLACK_WEBHOOK_URL env var to enable Slack notifications
```

## What Gets Tested

### GitHub Integration
- ✅ GitHub API connectivity and authentication
- ✅ PR fetching with date-based server-side filtering
- ✅ PR state detection (draft, approved, merged, closed)
- ✅ Review status analysis (approved, changes requested)
- ✅ Filtering dismissed reviews

### Jira Integration
- ✅ Jira API connectivity and authentication
- ✅ JQL search by PR URL in custom field
- ✅ Transition fetching via REST API
- ✅ Current ticket status retrieval
- ✅ Workflow navigation and path finding

### Multi-Repository Features
- ✅ Global batching across repositories
- ✅ Cross-repository PR aggregation
- ✅ Ticket deduplication
- ✅ "Most behind" PR state selection
- ✅ Repository context in output

### Transition Logic
- ✅ Direct transition matching
- ✅ Multi-step transition path finding
- ✅ Terminal state detection (skip Closed/Done)
- ✅ Already-in-correct-status detection
- ✅ State priority comparison

### Slack Integration
- ✅ Notification collection
- ✅ Ticket-first formatting with grouped PRs
- ✅ Cross-repository PR display
- ✅ Statistics tracking (tickets in correct status, unlinked PRs)
- ✅ Message sending (if webhook configured)

### Safety
- ❌ **Does NOT execute transitions** (read-only mode)
- ❌ **Does NOT modify Jira tickets** (safe to run anytime)

## Key Features Demonstrated

### Global Batching

When a Jira ticket has PRs across multiple repositories:

```
PROJ-123 has PRs in 3 repos:
  - myorg/backend PR #42 (merged)
  - myorg/frontend PR #15 (approved)  ← Most behind
  - myorg/api PR #7 (merged)

→ Ticket transitions based on "approved" state (most behind)
→ All PRs shown in Slack notification with repo context
```

### Multi-Step Transitions

When direct transition isn't available:

```
Ticket in "To Do" → Target "Dev Complete"
No direct path available

Attempting multi-step:
  Step 1: To Do → In Progress
  Step 2: In Progress → Code Review
  Step 3: Code Review → Dev Complete
```

### Terminal State Protection

```
PROJ-456: Current Status = Closed
⊗ Ticket is in terminal state - skipping
(Never auto-reopens or modifies closed tickets)
```

## Customization

Edit `test/jirasync/main.go` to:

- Change the time window (default: 7 days)
  ```go
  sinceTime := time.Now().AddDate(0, 0, -7)  // Change -7 to desired days
  ```

- Test different repositories
  ```go
  repos = []jirasync.Repository{
      {Owner: "myorg", Name: "myrepo"},
  }
  ```

- Switch between hardcoded and environment variables
  ```go
  ghToken := os.Getenv("GITHUB_TOKEN")  // Use env var
  // ghToken := "github_pat_..."  // Or hardcode for testing
  ```

- Enable/disable Slack notifications
  ```go
  slackWebhookURL := os.Getenv("SLACK_WEBHOOK_URL")  // Enable
  // slackWebhookURL := ""  // Disable
  ```

## Configuration File Format

```yaml
# test-repos.yaml
repositories:
  - owner: org1
    name: repo1
  - owner: org1
    name: repo2
  - owner: org2
    name: repo3
```

This allows testing the exact same multi-repository flow that production jirasync uses.

## Understanding the Output

### PR State Priority

When multiple PRs exist for one ticket, the test shows which state "wins":

```
Priority 1 (most behind): draft, changes_requested
Priority 2 (in review):   open, review_requested, approved
Priority 3 (complete):    merged, closed
```

Lower priority = more behind = determines ticket state.

### Transition Indicators

- `✓` Would execute transition (direct or multi-step)
- `⊗` Skipped (terminal state, no transitions available)
- `⚠` Warning (no direct path, but multi-step possible)
- `→` Indicates selected PR (most behind) or transition path

### Slack Preview

If webhook is configured, the test sends an actual Slack notification. Otherwise, it just shows:
```
Tip: Set SLACK_WEBHOOK_URL env var to enable Slack notifications
```

## Troubleshooting

### No PRs found

- Try a longer time window (change `-7` to `-30` days)
- Verify the repository has recent PR activity
- Check that PRs have been updated in the time window (not just created)

### JQL search fails

- Verify `JIRA_PR_FIELD` is the correct custom field ID
- Check if the field exists in your Jira project
- Ensure PRs are linked with full URLs (not just PR numbers)
- Test the JQL query in Jira's issue navigator

### No transitions found

- This usually means API token permissions issue
- Regenerate Jira API token with full permissions
- Check if your user has permission to view workflow for those tickets
- Review the "Available transitions" output to see what Jira returns

### Empty transitions array

- Create a new Jira API token with proper permissions
- Ensure token has "Browse Projects" and "Transition Issues" scopes
- Check if ticket is in a workflow state that has no outgoing transitions

### Slack notification fails

- Verify webhook URL is correct and active
- Check Slack app has permission to post to the channel
- Test webhook manually: `curl -X POST -H 'Content-Type: application/json' -d '{"text":"test"}' $SLACK_WEBHOOK_URL`
- Check for specific error in output

## Next Steps

After successful testing:

1. Update tokens to use environment variables instead of hardcoded values
2. Create a production `repos.yaml` config file
3. Run actual jirasync CLI: `jirasync --config repos.yaml --github-token $GITHUB_TOKEN ...`
4. Set up automated runs via GitHub Actions or cron
5. Monitor Slack notifications for visibility

## Differences from Production jirasync

| Feature | Test Harness | Production jirasync |
|---------|--------------|---------------------|
| **Executes transitions** | ❌ No (read-only) | ✅ Yes |
| **Uses config file** | ✅ Yes (optional) | ✅ Yes (required) |
| **Slack notifications** | ✅ Yes (if webhook set) | ✅ Yes (if webhook set) |
| **Multi-repo support** | ✅ Yes | ✅ Yes |
| **Global batching** | ✅ Yes | ✅ Yes |
| **Shows detailed output** | ✅ Very verbose | ✅ Standard logging |
| **Safe to run** | ✅ Always safe | ⚠️ Modifies Jira |

The test harness uses the exact same libraries and logic as production, just skips the final transition execution step.
