# ynewslib

Self-hosted, server-rendered Hacker News frontend

- Reads via the [official Firebase API](https://github.com/hackernews/api) (top/new/best/ask/show/jobs, full comment trees, users)
- Collapsible threads via native `<details>` (no JS)
- HTMX for vote buttons (stubs until M3)
- Voting/login proxied to news.ycombinator.com (M3, currently stubbed)

## Run

    go run .

Or with Docker:

    docker build -t ynewslib . && docker run -p 8080:8080 ynewslib

Config: `PORT` env var (default 8080).
