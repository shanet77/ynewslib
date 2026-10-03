# ynewslib

Self-hosted, server-rendered Hacker News frontend

- Reads via the [official Firebase API](https://github.com/hackernews/api) (top/new/best/ask/show/jobs, full comment trees, users)
- Collapsible threads via native `<details>` (no JS, no client dependencies at all)
- Comment HTML sanitized with bluemonday; strict CSP, no external requests

## Run

    go run .

Or with Docker:

    docker build -t ynewslib . && docker run -p 8080:8080 ynewslib

Config: `PORT` env var (default 8080).
