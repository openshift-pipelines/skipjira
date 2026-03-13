#!/bin/bash
# skipjira pre-push hook

while read local_ref local_sha remote_ref remote_sha; do
  branch="${remote_ref#refs/heads/}"

  # Check if new branch on remote
  if git ls-remote --heads origin "$branch" 2>/dev/null | grep -q "$branch"; then
    continue
  fi

  # Extract Jira ID from branch name
  jira_id=$(echo "$branch" | grep -oE '[A-Z]+-[0-9]+' | head -n1)

  # Prompt user
  if [ -n "$jira_id" ]; then
    echo -n "Jira ticket ID [$jira_id] (or 'skip' to skip): " >&2
  else
    echo -n "Jira ticket ID (or press Enter to skip): " >&2
  fi

  # Read from TTY 
  read input < /dev/tty

  # Handle skip
  if [ "$input" = "skip" ] || [ "$input" = "-" ]; then
    echo "Skipping Jira link for $branch" >&2
    continue
  fi

  final_id="${input:-$jira_id}"

  if [ -n "$final_id" ]; then
    # Queue for later processing
    skipjira queue --branch "$branch" --jira-ids "$final_id" 2>&1
    echo "✓ Queued: $branch → $final_id" >&2
  else
    echo "No Jira ID provided, skipping" >&2
  fi
done
