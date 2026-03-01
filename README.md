# kiro-relay

Remote control [kiro-cli](https://kiro.dev/cli/) from chat apps. Send messages from your phone, get responses from kiro-cli running on your laptop.

Inspired by [Claude Code Remote Control](https://code.claude.com/docs/en/remote-control).

## Supported platforms

- **Telegram** ✅
- LINE — planned
- WeChat — planned

## Setup

```bash
cp .env.example .env
# Edit .env with your values
```

| Variable | Required | Description |
|---|---|---|
| `KIRO_WORK_DIR` | No | Directory where kiro-cli runs. Default: `~` |
| `KIRO_TRUST_TOOLS` | No | `*` to trust all tools, comma-separated list for specific tools, or empty for none |
| `TELEGRAM_BOT_TOKEN` | No* | Bot token from [@BotFather](https://t.me/BotFather) |
| `TELEGRAM_ALLOWED_USER_IDS` | No* | Comma-separated Telegram user IDs |

*At least one platform must be fully configured. Telegram requires both `TELEGRAM_BOT_TOKEN` and `TELEGRAM_ALLOWED_USER_IDS`.

## Run

```bash
# From source
go run ./cmd/kiro-relay

# Or build and run
go build -o kiro-relay ./cmd/kiro-relay
./kiro-relay
```

## Usage

In Telegram:

1. `/start` — begin a new kiro session
2. Send any message — relayed to kiro-cli, response sent back
3. `/end` — end the session

## Project structure

```
cmd/kiro-relay/             entrypoint
internal/
  config/                   .env loading
  kiro/                     kiro-cli wrapper
  relay/
    platform.go             Platform interface
    session.go              shared session manager
    telegram/               Telegram adapter
```

## Adding a new platform

1. Create `internal/relay/<platform>/adapter.go` implementing `relay.Platform`
2. Add platform config to `internal/config/config.go`
3. Wire it up in `cmd/kiro-relay/main.go`

## Requirements

- Go 1.21+
- [kiro-cli](https://kiro.dev/cli/) installed and authenticated
