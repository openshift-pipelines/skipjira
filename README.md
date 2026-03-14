# SkipJira

Automatically link GitHub Pull Requests to Jira tickets without slowing down your workflow.

## How It Works

1. **Git Push** → Pre-push hook prompts for Jira ticket IDs (extracts from branch name by default)
2. **Queued** → Request is queued locally, push continues immediately
3. **Cron Job** → Batch processor finds the PR and updates Jira tickets in the background
4. **Done** → Jira tickets get updated with PR links automatically

**Benefits:**
- Non-blocking: Push completes instantly, Jira updates happen later
- Batch processing: Efficient background sync via cron
- Audit trail: Complete log of all operations in `~/.config/skipjira/audit.jsonl`

## Setup

### 1. Install Binary

```bash
go install github.com/theakshaypant/skipjira/cmd/skipjira@latest
```

This installs `skipjira` to `$GOPATH/bin` (usually `~/go/bin`). Make sure it's in your `$PATH`.

### 2. Configure Global Settings

```bash
skipjira config
```

You'll be prompted for:
- **GitHub token** (with repo access)
- **Jira URL** (e.g., `https://issues.company.com`) - Must support Jira v3 API
- **Jira token** (API token)
- **Jira PR field** (e.g., `customfield_12310220`)

These settings are saved to `~/.config/skipjira/config.yaml` and reused across all repositories.

**Note:** SkipJira uses the Jira REST API v3.

### 3. Install Pre-Push Hook

In each repository where you want to use skipjira:

```bash
cd /path/to/your/repository
skipjira install
```

This installs the pre-push hook to `.git/hooks/pre-push` and creates repo-specific config.

### 4. Setup Cron Job

```bash
crontab -e
```

Add this line to sync every 6 hours:
```
0 */6 * * * skipjira sync
```

The cron job processes queued entries and updates Jira tickets in the background. Adjust the schedule as needed (e.g., `*/30 * * * *` for every 30 minutes).

## Usage

### Normal Workflow

```bash
# 1. Create and push a branch
git checkout -b feature/PROJ-123-fix-bug
git push origin feature/PROJ-123-fix-bug

# 2. Hook prompts for Jira ID
Jira ticket ID [PROJ-123] (or 'skip' to skip):
# Press Enter to use auto-detected ID, or type custom IDs

# 3. Push completes immediately (queued for background processing)
✓ Queued feature/PROJ-123-fix-bug for Jira tickets: PROJ-123

# 4. Create your PR on GitHub (as usual)

# 5. Cron job updates Jira automatically (within 30 minutes)
```

### Options When Prompted

- **Press Enter**: Use auto-detected ID from branch name
- **Multiple IDs**: Type `PROJ-123,PROJ-456` (comma-separated)
- **Skip**: Type `skip`, `-`, or just press Enter if no ID detected

## Commands

- `skipjira config` - Configure global settings
- `skipjira install` - Install pre-push hook in repository
- `skipjira sync` - Process queue and update Jira (run via cron)
- `skipjira status` - Show current queue (pending/failed entries)
- `skipjira audit` - View complete audit log of all operations

## Configuration

### Global (`~/.config/skipjira/config.yaml`)

```yaml
github_token: ghp_xxxxx
jira_url: https://issues.company.com
jira_token: xxxxx
jira_pr_field: customfield_12345678
```

### Per-Repository (`.git/skipjira-config.yaml`)

```yaml
repo_owner: theakshaypant
repo_name: skipjira
# Optional: override global tokens/URLs if needed
```
