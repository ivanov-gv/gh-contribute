# gh-contribute

A GitHub CLI extension that lets AI agents interact with pull requests as real contributors — reading reviews, posting comments, and leaving reactions.

## TL;DR

```bash
# install
gh extension install ivanov-gv/gh-contribute

# see PR details (auto-detects from current branch)
gh contribute pr

# list all comments and reviews on a PR
gh contribute comments

# post a comment
gh contribute comment "Fixed the issue, please re-review"

# react to a comment
gh contribute react 123456789 eyes --type issue
gh contribute react 987654321 rocket --type review

# show inline comments for a specific review
gh contribute review 3929204495
```

All commands auto-detect the repository (from git remote) and PR number (from current branch). Run `gh contribute auth login` once to authenticate.

---

## Why

AI coding agents (Claude Code, Copilot, Cursor, etc.) can write code, commit, and push — but they can't participate in the review process on GitHub. They have no way to:

- Read what reviewers said about their PR
- Acknowledge comments with reactions
- Reply to feedback
- Show progress on addressing review comments

**gh-contribute** bridges this gap. It gives agents a simple CLI interface to the GitHub review workflow, turning them from "push and forget" tools into active PR participants.

## Use Cases

### Remote control through GitHub reviews

A typical agent workflow today:

1. Agent finishes work, commits, pushes, opens a PR
2. **Dead end** — the agent has no idea what happens next

With gh-contribute:

1. Agent finishes work, commits, pushes, opens a PR
2. A reviewer leaves comments and suggestions on the PR
3. Something triggers the agent again (webhook, polling, slash command)
4. Agent runs `gh contribute comments` to read all review feedback
5. Agent addresses each comment, pushes fixes
6. Agent runs `gh contribute comment "Addressed all feedback, PTAL"`
7. Repeat until merged

The entire interaction happens through GitHub — no need to access the agent's terminal or UI.

### Live status through reactions

When an agent is processing review comments, nobody on GitHub knows what's happening. With reactions, the agent can broadcast its progress:

1. Agent receives notification about new review comments
2. Runs `gh contribute comments` to get the list
3. For each comment, the agent:
   - Adds 👀 (`eyes`) reaction — "I'm looking at this"
   - Works on the fix
   - Adds 🚀 (`rocket`) reaction — "Done"
4. When all comments are addressed, posts a summary comment

Everyone watching the PR sees real-time status without leaving GitHub.

### Automated triage and acknowledgment

An agent can periodically check for new comments across PRs and:

- React with 👍 to acknowledge simple suggestions
- React with 😕 (`confused`) to flag comments it doesn't understand
- Post clarifying questions as replies
- Prioritize comments based on reviewer authority

## Commands

### `gh contribute pr`

Show details about a pull request in human-readable markdown.

```bash
# auto-detect PR from current branch
gh contribute pr

# specify PR number explicitly
gh contribute pr 42
```

Output:
```
# test-pr: test gh extension #1
open, by @ivanov-gv, 1 commit `test-pr` -> `main`, no merge conflict
https://github.com/ivanov-gv/gh-contribute/pull/1

Reviewers:
Assignees: @ivanov-gv
Labels:
Projects:
Milestone:
Issues:

Conversation: 1 comment

---

test description

---
```

### `gh contribute comments`

List issue comments and reviews on a pull request. Shows reactions with "by you" tracking, hides minimized comments and fully-resolved reviews.

```bash
# all comments and reviews
gh contribute comments

# specify PR
gh contribute comments --pr 42
```

Output:
```
issue #4038597073 by you (@ivanov-gv-ai-helper)
_2026-03-11 11:33:27_

test comment from gh-contribute 🚀

(1 🚀)
reactions by you: (1 🚀)

---
issue #4038819817 by @ivanov-gv
_2026-03-11 12:15:54_

> test comment from gh-contribute 🚀
test reply

(1 😕)
reactions by you:

---
review #3929204495 by @ivanov-gv
_2026-03-11 12:17:34_

submit review

comments: 3
(1 👀)
reactions by you:

---
review #3929353771 by @ivanov-gv | hidden: Resolved
```

Key features:
- **Issue comments** show id, author, date, body, and reactions
- **Reviews** show id, author, date, body, inline comment count, and reactions
- **"reactions by you"** tracks which reactions belong to the authenticated user (works with GitHub App `[bot]` accounts)
- **Hidden items**: minimized issue comments and reviews (`isMinimized: true`) show only the header line with the reason
- Review inline comments are not expanded — use the review id for detailed inspection

### `gh contribute comment`

Post a top-level comment on a pull request.

```bash
gh contribute comment "All review comments have been addressed. Ready for re-review."

gh contribute comment --pr 42 "Automated analysis complete. Found 3 potential issues."
```

### `gh contribute react`

Add a reaction to a comment. Use the comment id from the `comments` output.

```bash
# react to a review comment (default)
gh contribute react 123456789 rocket

# react to a top-level (issue) comment
gh contribute react 123456789 eyes --type issue
```

Valid reactions: `+1`, `-1`, `laugh`, `confused`, `heart`, `hooray`, `rocket`, `eyes`

### `gh contribute review`

Show a specific review's inline comments with thread context.

```bash
# show inline comments for review by id (use id from comments output)
gh contribute review 3929204495

# specify PR explicitly
gh contribute review 3929204495 --pr 42
```

Output shows the review body, reactions, and all inline comment threads grouped and sorted, with replies indented using `>`.

## Installation

### From GitHub releases

```bash
gh extension install ivanov-gv/gh-contribute
```

### From source

```bash
git clone https://github.com/ivanov-gv/gh-contribute.git
cd gh-contribute
go build -o gh-contribute ./cmd/gh-contribute
```

Then either:
- Add the binary to your `PATH`, or
- Symlink it into `~/.local/share/gh/extensions/gh-contribute/`

### Authentication

gh-contribute uses its own GitHub App and issues tokens via the **Device Authorization Flow** (RFC 8628). API calls appear as `myapp[bot] on behalf of <username>`, giving them proper attribution without exposing your personal token.

#### First-time login

```bash
gh contribute auth login
```

This prints a URL and a short code, then waits:

```
Open: https://github.com/login/device
Enter code: XXXX-YYYY
```

Open the URL in your browser, enter the code, and authorize the app. The token is stored at `~/.config/gh-contribute/token` with `0600` permissions and reused for all future sessions.

#### Check status

```bash
gh contribute auth status
# Logged in as: your-username
```

#### CI and non-interactive environments

Device Flow requires a browser. In CI, set the `GH_CONTRIBUTE_TOKEN` environment variable instead:

```bash
export GH_CONTRIBUTE_TOKEN=github_pat_...
```

When set, the env var takes priority over the stored token file. No login step needed.

#### Token lifecycle

Tokens are non-expiring by default (GitHub App expiration is disabled). If a token is revoked or a request returns `401`, gh-contribute exits with:

```
Error: token invalid or expired — run 'gh contribute auth login' to reauthenticate
```

## Auto-detection

When `--pr` is not specified, gh-contribute automatically:

1. Reads the current git branch name
2. Searches for an open PR with that branch as the head
3. Uses the first match

When the repository is not specified (it never needs to be), gh-contribute:

1. Reads the `origin` remote URL from git
2. Parses the owner and repo name from it (supports both SSH and HTTPS remotes)

This means in most cases you just run `gh contribute comments` with zero flags and it does the right thing.

## Project Structure

```
gh-contribute/
├── cmd/gh-contribute/main.go           # entry point
├── internal/
│   ├── auth/                           # Device Authorization Flow (RFC 8628)
│   │   ├── auth.go                     # RunDeviceFlow, GetUsername (via go-github SDK)
│   │   └── errors.go                   # ErrTokenInvalid sentinel
│   ├── cmd/                            # cobra command definitions
│   │   ├── root.go                     # root command, dependency wiring, PersistentPreRunE
│   │   ├── auth.go                     # auth login / auth status commands
│   │   ├── pr.go                       # pr command + PR auto-detection
│   │   ├── comments.go                 # comments command
│   │   ├── comment.go                  # comment command (post)
│   │   ├── react.go                    # react command
│   │   └── review.go                   # review command (inline comment detail)
│   ├── config/
│   │   ├── config.go                   # repo detection from git remote
│   │   └── token.go                    # LoadToken / SaveToken, GH_CONTRIBUTE_TOKEN env, ErrNotAuthenticated
│   ├── format/format.go                # shared formatting utilities (reactions, dates, authors)
│   ├── github/graphql.go               # GraphQL client (queries)
│   ├── git/git.go                      # git helpers (current branch)
│   └── service/
│       ├── pr/
│       │   ├── pr.go                   # PR info and lookup via GraphQL
│       │   └── format.go               # PR markdown formatting
│       ├── comment/
│       │   ├── comment.go              # list via GraphQL, post via REST
│       │   └── format.go               # comment/review markdown formatting
│       ├── reaction/reaction.go        # add reactions via REST
│       └── review/
│           ├── review.go               # review detail with inline comments via GraphQL
│           └── format.go               # review detail markdown formatting
├── .claude/
│   ├── hooks/session-start.sh          # SessionStart hook: build + auth check
│   └── settings.json                   # Claude Code hook registration
├── go.mod
└── go.sum
```

Built with:
- [google/go-github](https://github.com/google/go-github) — GitHub REST API client (mutations + `GetUsername`)
- GitHub GraphQL API v4 — for rich read queries (reactions, review threads, metadata)
- [spf13/cobra](https://github.com/spf13/cobra) — CLI framework
- [joho/godotenv](https://github.com/joho/godotenv) — `.env` file loading
- [rs/zerolog](https://github.com/rs/zerolog) — structured logging

### Claude Code on the web

The `.claude/hooks/session-start.sh` hook runs automatically at the start of every remote Claude Code session. It:

1. Runs `go mod download` to warm the module cache
2. Builds the extension binary
3. Checks `auth status` — if no token is found, starts `auth login` so you can authorize before the agent begins work

This ensures the agent always has valid GitHub credentials before it needs them, preventing mid-task authentication interruptions.

## Ways to Improve

- **Reply to review comments** — post threaded replies to specific inline comments
- **Diff-aware comments** — post inline review comments on specific files and lines
- **Webhook listener** — built-in server that watches for review events and triggers agent actions
- **Multi-PR support** — list and manage comments across all open PRs in a repo
- **Token refresh** — currently tokens are non-expiring; if GitHub App expiration is enabled, add a refresh flow
