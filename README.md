# ynewslib

Self-hosted, server-rendered Hacker News frontend

View a live version at https://hackernews.hoopnerd.xyz/

- Reads via the [official Firebase API](https://github.com/hackernews/api) (top/new/best/ask/show/jobs, full comment trees, users)
- Collapsible threads via native `<details>` (no JS, no client dependencies at all)
- Theming: `auto`/`dark`/`light` presets in the nav, or any custom palette via `/theme/custom?bg=14171a&fg=d6d3cd&dim=8a8f98&link=f0a35e&line=2a2f36&accent=ff6600` (hex, all vars optional, remembered in a cookie)

## Run

    go run .

Or with Docker:

    docker build -t ynewslib . && docker run -p 8080:8080 ynewslib

Config: `PORT` env var (default 8080).
