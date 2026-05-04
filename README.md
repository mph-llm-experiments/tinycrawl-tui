# tinycrawl

A terminal dungeon crawler using Cairn 2e rules. Generate characters, explore procedural dungeons, and survive — all from your terminal.

<!-- screenshot placeholder -->

## Features

- Offline-capable play with a local content pack
- Kitty terminal graphics protocol for images
- Creative action resolution via AI game master
- Expedition mode (full dungeon runs) and sprint mode (single-session delves)

## Install

**Homebrew:**
```sh
brew install mph-llm-experiments/tap/tinycrawl
```

**Go install:**
```sh
go install github.com/mph-llm-experiments/tinycrawl-tui@latest
```

**Binary download:** grab a release from [GitHub Releases](https://github.com/mph-llm-experiments/tinycrawl-tui/releases).

## Quick Start

```sh
tinycrawl
```

## Config

Optional config file at `~/.tinycrawl/config.toml`:

| Field | Default | Description |
|-------|---------|-------------|
| `server` | `https://tinycrawl.puddingtime.net` | Game server URL |
| `passphrase` | — | Server passphrase |
| `anthropic_key` | — | Anthropic API key for AI GM |
| `default_pack` | `the-dark-below` | Content pack to use |
| `default_type` | `sprint` | Default game type (`sprint` or `expedition`) |
| `image_mode` | `auto` | Image rendering (`auto`, `kitty`, `none`) |
