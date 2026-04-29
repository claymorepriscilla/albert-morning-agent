# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run the agent locally (requires .env)
go run ./cmd/agent

# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out

# Run a single package's tests
go test ./internal/gemini/...

# Lint (must pass before merging — same as CI)
golangci-lint run --timeout=5m

# Build binary
go build ./cmd/agent
```

## Architecture

This is a single-binary Go application with no HTTP server — it runs once, does its work, and exits. Designed to be invoked by a scheduler (GitHub Actions cron or local cron).

**Flow:**
1. `config.Load()` — reads env vars (`.env` for local dev, OS env for CI)
2. `cmd/agent/main.go` — orchestrates everything; switches on `LOTTERY_ONLY` env var
3. For each topic: `news.FetchRSS` → `gemini.Summarize` → `line.Send`
4. Gold topic uses `gold.FetchPrice` (Thai Gold API) + RSS headlines → combined message

**Package responsibilities:**

| Package | Role |
|---|---|
| `internal/config` | Load + validate env vars; fails fast if required vars missing |
| `internal/news` | Parse Google News RSS via `gofeed`; filters to last 24h only |
| `internal/gemini` | Wraps Groq API (OpenAI-compatible) with `llama-3.3-70b-versatile` |
| `internal/gold` | Fetches Thai gold prices from `api.chnwt.dev`; formats combined message |
| `internal/line` | Sends LINE messages — Push (single user) or Broadcast (all followers) |

**Key design decisions:**
- **Best-effort per topic** — errors are logged and skipped; one failed topic doesn't abort others
- **Lottery detection via RSS** — no hardcoded dates; `processIfNewsFound` sends only when RSS has recent content
- **`LOTTERY_ONLY=true`** — set automatically by the 08:00 UTC cron run in the workflow; controls which briefings run
- **`internal/gemini` package name is misleading** — it actually calls the Groq API, not Google Gemini

## Testing Patterns

Tests use `export_test.go` files to expose internal vars (e.g., `groqEndpoint`, `apiURL`, `pushURL`) for override in tests — allowing HTTP mocking without a real server. See `internal/gemini/export_test.go` as the pattern.

## Environment Variables

| Var | Required | Notes |
|---|---|---|
| `GROQ_API_KEY` | Always | Groq Console key |
| `LINE_CHANNEL_ACCESS_TOKEN` | Always | LINE Messaging API token |
| `LINE_USER_ID` | Push mode only | Must start with `U` |
| `LINE_BROADCAST` | No | `"true"` = broadcast to all followers |
| `LOTTERY_ONLY` | No | `"true"` = run only lottery briefing (afternoon run) |

## CI / GitHub Actions

- Two scheduled triggers: `0 0 * * *` (morning) and `0 8 * * *` (afternoon lottery check)
- The `Detect run mode` step sets `LOTTERY_ONLY` based on UTC hour — no manual input needed
- `golangci-lint v1.58.0` runs on every CI job; lint must pass before merge
- `FORCE_JAVASCRIPT_ACTIONS_TO_NODE24=true` is set at job level for Node.js 24 compatibility
