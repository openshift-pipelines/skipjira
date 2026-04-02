# Quick Setup Guide for JiraSync GitHub Actions

Follow these steps to set up the automated jirasync workflow.

## Step 1: Enable GitHub Actions

1. Go to your repository: `https://github.com/openshift-pipelines/skipjira`
2. Click `Settings` → `Actions` → `General`
3. Under "Actions permissions", select "Allow all actions and reusable workflows"
4. Click `Save`

## Step 2: Configure Required Secrets

### GitHub Token

1. Go to GitHub Settings: `https://github.com/settings/tokens`
2. Click "Generate new token (classic)"
3. Name: `JiraSync Bot Token`
4. Select scope: `repo` (full control of private repositories)
5. Click "Generate token"
6. Copy the token (starts with `ghp_`)

Now add it to your repository:
1. Go to `Settings` → `Secrets and variables` → `Actions`
2. Click "New repository secret"
3. Name: `JIRASYNC_GITHUB_TOKEN`
4. Value: Paste the token
5. Click "Add secret"

### Jira Credentials

Get your Jira API token:
1. Go to: `https://id.atlassian.com/manage-profile/security/api-tokens`
2. Click "Create API token"
3. Label: `JiraSync Bot`
4. Copy the token

Add Jira secrets to GitHub:

| Secret Name | Where to Find | Example Value |
|------------|---------------|---------------|
| `JIRA_URL` | Your Jira instance URL | `https://issues.redhat.com` |
| `JIRA_EMAIL` | Your Jira login email | `yourname@company.com` |
| `JIRA_TOKEN` | Token from step above | `ATBBxxxxxxxxxxxxxxxx` |
| `JIRA_PR_FIELD` | See instructions below | `customfield_12310220` |

**Finding JIRA_PR_FIELD:**
1. Open any Jira ticket in browser
2. Add `?expand=names` to URL
3. Search for "Git PR URL" or similar field
4. Copy the field ID (e.g., `customfield_12310220`)

## Step 3: Configure Optional Features

### Release Notes (Recommended)

**Jira Fields:**

To find field IDs:
```bash
curl -u yourname@company.com:YOUR_JIRA_TOKEN \
  https://issues.redhat.com/rest/api/3/field | jq '.[] | select(.name | contains("Release"))'
```

Add these secrets:
- `JIRA_RELEASE_NOTES_TEXT_FIELD` - Field for release notes text
- `JIRA_RELEASE_NOTES_TYPE_FIELD` - Field for type (Bug Fix/Feature/Enhancement)
- `JIRA_RELEASE_NOTES_STATUS_FIELD` - Field for status (Proposed/Approved)

**Gemini AI (for auto-generation):**

1. Go to: `https://aistudio.google.com/app/apikey`
2. Create API key
3. Add secret: `GEMINI_API_KEY`
4. (Optional) Add secret: `GEMINI_MODEL` with value like `gemini-2.0-flash-exp`

### Slack Notifications (Recommended)

1. Go to your Slack workspace
2. Create incoming webhook: `https://api.slack.com/messaging/webhooks`
3. Choose a channel (e.g., `#jira-sync`)
4. Copy webhook URL
5. Add secret: `SLACK_WEBHOOK_URL`

## Step 4: Verify Setup

### Option 1: Manual Test Run

1. Go to `Actions` tab in GitHub
2. Select "JiraSync - Automated PR to Jira Synchronization"
3. Click "Run workflow"
4. Leave inputs empty (uses defaults)
5. Click "Run workflow" button
6. Watch the logs for any errors

### Option 2: Wait for Scheduled Run

The workflow will automatically run nightly at 1 AM UTC.

## Step 5: Verify It's Working

### Check Logs

After a run completes:
1. Go to `Actions` tab
2. Click the latest "JiraSync" run
3. Click the `jirasync` job
4. Review the output:
   - Should show PRs found
   - Should show tickets transitioned
   - Look for any errors or warnings

### Expected Output

```
=== Collecting PRs from openshift-pipelines/skipjira ===
  Found 5 PRs updated since 2026-04-01
  PR #123 (merged) → target status: On QA
    ✓ Linked to 1 ticket(s): [SRVKP-12345]

=== Processing Tickets (Global Batching) ===
Found 3 unique tickets across all repositories

Processing SRVKP-12345 (current: 'Code Review')
  PR #123 state: merged → Jira target: 'On QA'
  ✓ Transitioned SRVKP-12345: 'Code Review' → 'On QA' (PR state: merged)

Jirasync completed - 1 tickets transitioned
```

### Check Slack (if configured)

You should receive a message like:
```
The following Jira tickets were processed for workflow transitions.

SRVKP-12345 `Code Review` → `On QA` ✅ APPLIED
  • openshift-pipelines/skipjira#123: Fix bug in sync logic

🤖 jirasync | 1 tickets processed • 2 already in correct status • 0 unlinked PRs
```

## Troubleshooting

### "Secret not found" errors

- Verify secret names are exact matches (case-sensitive)
- Check secrets are added at repository level, not organization level

### "Authentication failed"

- GitHub token: Verify it has `repo` scope
- Jira: Verify email/token combination works:
  ```bash
  curl -u email@company.com:TOKEN https://issues.redhat.com/rest/api/3/myself
  ```

### "No PRs found"

- Default looks at yesterday's PRs
- Try manual run with `--since 2026-03-01` to see more PRs

### "No tickets transitioned"

Common causes:
1. PRs not linked to Jira (check logs for "No Jira tickets linked")
2. Tickets already in correct status (check logs for "Already in target status")
3. No transitions available (check Jira workflow permissions)

## Next Steps

1. ✅ Verify workflow runs successfully
2. ✅ Check Jira tickets are being updated
3. ✅ Monitor Slack notifications (if enabled)
4. 📝 Adjust cron schedule if needed (see [README.md](./README.md#change-schedule))
5. 📝 Add more repositories to `jirasync-repos.yaml` if needed

## Getting Help

- Review logs in Actions tab
- Check [main README](./README.md) for detailed documentation
- Review [jirasync documentation](../../cmd/jirasync/README.md)
- Create an issue in this repository

## Security Checklist

- [ ] All secrets configured (no hardcoded values)
- [ ] GitHub token has minimal required scope (`repo`)
- [ ] Jira API token used (not password)
- [ ] Secrets not visible in logs
- [ ] Webhook URLs kept private
- [ ] Regular token rotation scheduled
