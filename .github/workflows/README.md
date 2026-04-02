# GitHub Actions Workflow - JiraSync

This directory contains the automated workflow for running jirasync nightly to synchronize GitHub PRs with Jira tickets.

## Workflow: jirasync.yml

**Triggers:**
- **Scheduled:** Runs nightly at 1 AM UTC
- **Manual:** Can be triggered manually via "Actions" tab with optional parameters

**Features:**
- Automatic PR state → Jira status transitions
- Release notes extraction and AI generation (with Gemini)
- Slack notifications for transitions
- Cross-repository support
- Error handling and logs

## Required Secrets

Configure these secrets in your repository settings (`Settings` → `Secrets and variables` → `Actions`):

### GitHub Authentication

| Secret Name | Description | Example |
|------------|-------------|---------|
| `JIRASYNC_GITHUB_TOKEN` | GitHub Personal Access Token with `repo` scope | `ghp_xxxxxxxxxxxx` |

### Jira Authentication (Required)

| Secret Name | Description | Example |
|------------|-------------|---------|
| `JIRA_URL` | Your Jira base URL | `https://issues.redhat.com` |
| `JIRA_EMAIL` | Email for Jira API authentication | `bot@company.com` |
| `JIRA_TOKEN` | Jira API token | `ATxxxxxxxxxxxxxxxx` |
| `JIRA_PR_FIELD` | Jira custom field ID for PR links | `customfield_12310220` |

### Jira Release Notes Fields (Optional)

| Secret Name | Description | Required |
|------------|-------------|----------|
| `JIRA_RELEASE_NOTES_TEXT_FIELD` | Field ID for release notes text | Optional |
| `JIRA_RELEASE_NOTES_TYPE_FIELD` | Field ID for release notes type (Bug Fix/Feature/Enhancement) | Optional |
| `JIRA_RELEASE_NOTES_STATUS_FIELD` | Field ID for release notes status (Proposed/Approved) | Optional |

### Gemini AI (Optional - for AI-generated release notes)

| Secret Name | Description | Required |
|------------|-------------|----------|
| `GEMINI_API_KEY` | Google Gemini API key | Optional |
| `GEMINI_MODEL` | Gemini model name (defaults to `gemini-2.0-flash-exp`) | Optional |

### Slack Notifications (Optional)

| Secret Name | Description | Required |
|------------|-------------|----------|
| `SLACK_WEBHOOK_URL` | Slack incoming webhook URL | Optional |

## How to Add Secrets

1. Go to your repository on GitHub
2. Click `Settings` → `Secrets and variables` → `Actions`
3. Click `New repository secret`
4. Add each secret with the exact name from the tables above

## Manual Workflow Trigger

You can manually trigger the workflow with custom parameters:

1. Go to `Actions` tab in GitHub
2. Select "JiraSync - Automated PR to Jira Synchronization"
3. Click "Run workflow"
4. Optional: Enter a custom date (e.g., `2026-03-01` or `01/03/2026`)
5. Click "Run workflow" button

## Monitoring

### View Logs

- Go to `Actions` tab
- Click on the latest workflow run
- Click on the `jirasync` job to see detailed logs

### Slack Notifications

If configured, you'll receive Slack notifications for:
- Successfully transitioned tickets (with PR details)
- Workflow failures

### Artifacts

On failure, logs are automatically uploaded as artifacts for 7 days.

## Workflow Configuration

### Change Schedule

Edit `.github/workflows/jirasync.yml` and modify the cron expression:

```yaml
schedule:
  - cron: '0 1 * * *'  # Runs at 1 AM UTC daily
```

Examples:
- `0 9 * * *` - 9 AM UTC daily
- `0 */6 * * *` - Every 6 hours
- `0 1 * * 1` - 1 AM UTC every Monday

### Change Date Range

By default, jirasync processes PRs from yesterday. To change:

1. Manual run: Use the `since` input parameter
2. Modify the workflow: Add `--since` flag in the workflow file

## Troubleshooting

### No tickets transitioned

**Check:**
1. Are PRs within the date range? (View logs for "Found X PRs")
2. Are PRs linked to Jira tickets? (Look for "No Jira tickets linked")
3. Are tickets already in correct status? (Check summary output)

### Authentication errors

**Verify:**
- All required secrets are set correctly
- GitHub token has `repo` scope
- Jira token has proper permissions
- Jira URL is correct (no trailing slash)

### Release notes not generated

**Verify:**
- `GEMINI_API_KEY` secret is set
- PR is in merged state (release notes only for merged PRs)
- Check Gemini API quota limits

### Workflow not running

**Check:**
- Workflow file is in `.github/workflows/` directory
- YAML syntax is valid
- Repository has Actions enabled (`Settings` → `Actions` → `General`)

## Security Best Practices

1. **Never commit secrets** - Always use GitHub Secrets
2. **Use dedicated tokens** - Create service accounts for automation
3. **Rotate tokens regularly** - Update secrets periodically
4. **Limit token scope** - GitHub token needs only `repo` access
5. **Monitor usage** - Check Action logs regularly

## Cost Considerations

### GitHub Actions

- Free tier: 2,000 minutes/month for private repos
- This workflow uses ~5-10 minutes per run
- Daily runs = ~150-300 minutes/month

### Gemini API

- Check your API quotas (typically 1500 requests/day, 15/min)
- One request per merged PR without release notes
- Monitor usage in Google Cloud Console

## Support

For issues or questions:
- Create an issue in this repository
- Check logs in Actions tab
- Review [jirasync documentation](../../cmd/jirasync/README.md)
