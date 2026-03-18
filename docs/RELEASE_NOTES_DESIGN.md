# Release Notes Feature - Internal Design Doc

## Overview

Automatically extracts or AI-generates release notes from PRs and adds them as Jira comments.

## Architecture

### Core Components

**1. Orchestrator** (`internal/releasenotes/generator.go`)
- Entry point: `GetOrGenerate() → Result{Notes, IsGenerated}`
- Tries extraction first, falls back to AI

**2. Extractor** (`internal/releasenotes/extractor.go`)
- Parses PR description for "Release Notes" section
- Supports: `## Release Notes`, `### Release Notes`, `Release Notes:`, `**Release Notes:**`
- Case-insensitive, removes code fences

**3. AI Generator** (`internal/gemini/`)
- Uses Gemini API (gemini-3-flash-preview by default)
- Context: PR title, description (2000 chars), diff (3000 chars), commits
- Configurable via `--gemini-model`

**4. Jira Comment** (`internal/jira/comments.go`)
- `AddReleaseNotesComment()` creates ADF-formatted comments
- Two panel types: info (blue) vs warning (orange)

## Two Execution Paths

### Path 1: Extracted (Blue Panel)
- **Trigger**: PR description has "Release Notes" section
- **Process**: Extract → Clean → Add blue info panel to Jira
- **Comment**: 📝 header, notes content, metadata (Type/Status), italic footer
- **No @mention, no AI call**

### Path 2: AI-Generated (Orange Panel)
- **Trigger**: No release notes in PR description
- **Process**: Fetch PR context → Gemini API → Get assignee → Add orange warning panel
- **Comment**: 🤖 header, AI notes, metadata, italic footer, @mention with review request
- **Assignee notified**

## Edge Cases Handled

1. **Multiple PR formats**: Supports `## Release Notes`, `### Release Notes`, `Release Notes:`, case-insensitive
2. **Code fences**: Strips ``` markers from extracted notes
3. **Missing release notes**: Falls back to AI generation
4. **Missing assignee**: Adds comment without @mention (graceful degradation)
5. **Multiple tickets per PR**: Same notes added to all linked tickets
6. **Empty PR description**: AI uses title + diff + commits
7. **Gemini API failures**: Logs warning, continues sync (non-blocking)
8. **Jira API failures**: Logs warning, continues to next ticket (non-blocking)
9. **Large diffs**: Truncates to 3000 chars (description: 2000 chars)
10. **No Gemini key**: Only extraction works, feature disabled if no notes found
11. **Duplicate runs**: Currently adds new comment each time (TODO: dedup)

## Usage

```bash
jirasync \
  --config <path-to-repos.yaml> \
  --github-token <token> \
  --jira-url <https://your-jira.com> \
  --jira-email <email@company.com> \
  --jira-token <jira-api-token> \
  --jira-pr-field <customfield_xxxxx> \
  --gemini-api-key <gemini-key> \
  --gemini-model gemini-3-flash-preview \
  --since 2024-01-01
```

**Or using environment variables:**
```bash
export GITHUB_TOKEN=ghp_xxx
export JIRA_TOKEN=xxx
export JIRA_URL=https://your-jira.com
export JIRA_EMAIL=user@company.com
export JIRA_PR_FIELD=customfield_xxxxx
export GEMINI_API_KEY=xxx

jirasync --config repos.yaml --since 2024-01-01
```

## Configuration

### Required Parameters
- `--config`: Path to repositories YAML config file
- `--github-token`: GitHub personal access token
- `--jira-url`: Jira base URL
- `--jira-email`: Jira email for authentication
- `--jira-token`: Jira API token
- `--jira-pr-field`: Jira custom field ID for PR links (e.g., customfield_12345)

### Optional Parameters
- `--gemini-api-key`: Google Gemini API key (enables AI generation)
- `--gemini-model`: Gemini model to use (default: gemini-3-flash-preview)
- `--jira-release-notes-field`: Jira field ID for release notes (default: customfield_12317313)
- `--since`: Only process PRs updated since date (format: 2006-01-02 or DD/MM/YYYY, default: yesterday)
- `--slack-webhook`: Slack webhook URL for notifications (optional)

### Repository Config File (repos.yaml)
```yaml
repositories:
  - owner: your-org
    name: repo-1
  - owner: your-org
    name: repo-2
```

## Flow

```
PR Sync → Get PRs → For each PR:
  ├─ Find Jira tickets (JQL)
  ├─ If Gemini configured:
  │   ├─ Try extract → Found? → Blue panel
  │   └─ Not found? → AI generate (fetch diff/commits) → Get assignee → Orange panel + @mention
  └─ Add comment to Jira (ADF)
```

## ADF Format

**Panel types**: `info` (blue, extracted) | `warning` (orange, AI-generated)
**Text marks**: `strong` (headers) | `em` (footers)
**Mention**: `{type: "mention", attrs: {id: accountId}}`

## Error Handling

**Non-blocking** (log warning, continue): AI failures, comment failures, assignee fetch failures
**Blocking** (fail fast): Invalid credentials, missing config file

## Future Enhancements

1. **Duplicate prevention**: Check existing comments before adding
2. **Field updates**: Update Jira fields directly (when field IDs confirmed)
3. **Customizable metadata**: Configurable type/status values
4. **Templates**: Per-repo custom templates
5. **Multi-model support**: OpenAI, Claude fallback

## Testing

```bash
# Test extracted notes
cd test/jirasync && go run . <owner> <repo>

# Test AI generation
# Remove "Release Notes" from PR description, then run

# Test without Gemini
export GEMINI_API_KEY="" && go run ./cmd/jirasync --config ...
```

## Security & Performance

**Security**: Never commit API keys, use env vars. PR diffs sent to Gemini API (3000 char limit).

**Rate Limits**:
- GitHub: 5000/hr, 3-4 reqs/PR
- Jira: Varies, 2-3 reqs/ticket
- Gemini: 1500/day, 15/min, 1 req/PR (if AI used)

**Runtime**: ~30s for 10 PRs, ~2min for 50 PRs, ~5-10min for 200 PRs
