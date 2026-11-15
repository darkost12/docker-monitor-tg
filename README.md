# docker-monitor-tg

A small Telegram bot, written in Go, that lets you check on the Docker containers of a machine from your phone:
list the running containers and read the last lines of any container's logs.

Access is restricted to **one Telegram user**. Messages from anyone else are ignored without a reply.

## Features

- List running containers (ID, name, image, status).
- Show the last *N* log lines of a container, by name or by ID/hash.
- Single-user access control by numeric Telegram user ID.
- Cleans logs for Telegram: terminal color codes (ANSI escape sequences) are stripped, HTML is escaped, and long output is trimmed to the newest lines so it fits into one message.
- Safe `docker` invocation: no shell involved, container names are validated, and every call has a timeout.
- Friendly messages for the most common errors (container not found, Docker daemon not running).
- Graceful shutdown on `SIGINT`/`SIGTERM`.

## Commands

| Command | Description |
|---|---|
| `/ps` (or `/containers`) | List running containers |
| `/logs <name\|id> [N]` | Last `N` log lines (default 50, maximum 1000) |
| `/help` (or `/start`) | Show help |
| *plain text* | Same as `/logs`: send a container name or ID, optionally followed by `N` |

Examples:

```
/ps
/logs nginx
/logs 3f2a9c1b7d4e 200
web 30
```

## Requirements

- A Telegram bot token from [@BotFather](https://t.me/BotFather).
- Your numeric Telegram user ID (you can get it from [@userinfobot](https://t.me/userinfobot)).
- Docker, with access to the Docker daemon of the machine you want to monitor.
- To run without Docker: Go 1.21+ and the `docker` CLI in your `PATH`.

## Configuration

Settings are read from environment variables. A `.env` file in the working directory is loaded automatically; variables that are already set in the environment take precedence over it.

| Variable | Required | Description |
|---|---|---|
| `TELEGRAM_BOT_TOKEN` | yes | Bot token from @BotFather |
| `ALLOWED_USER_ID` | yes | Numeric ID of the only user who may use the bot |
| `HTTPS_PROXY` | no | Proxy for reaching `api.telegram.org` when Telegram is not directly accessible (e.g. `http://127.0.0.1:10801`) |

```bash
cp .env.example .env
# then edit .env
```

## Quick start (Docker Compose)

```bash
git clone https://github.com/darkost12/docker-monitor-tg.git
cd docker-monitor-tg

cp .env.example .env        # fill in TELEGRAM_BOT_TOKEN and ALLOWED_USER_ID
docker compose up -d --build

docker compose logs -f bot  # expect: "bot started as @your_bot"
```

Then send `/ps` to your bot. The bot's own container will appear in the list too.

To stop and remove it: `docker compose down`.

### How the container reaches the Docker daemon

The bot calls the `docker` CLI, so the image contains `docker-cli` and the host's Docker socket is mounted into the container:

```yaml
volumes:
  - /var/run/docker.sock:/var/run/docker.sock
```

The daemon stays on the host, so `docker ps` inside the container shows the host's containers.

### Without Compose

```bash
docker build -t docker-tg-bot .

docker run -d --name docker-tg-bot --restart unless-stopped \
  --env-file .env \
  -v /var/run/docker.sock:/var/run/docker.sock \
  docker-tg-bot
```

## Run locally (without a container)

```bash
cp .env.example .env
go run ./cmd/bot
```

Or build a binary:

```bash
go build -o bot ./cmd/bot
./bot
```

## Deploy on a server straight from GitHub

Because the repository is public, a server does not need a clone of it. Keep only a `docker-compose.yml` and a `.env` on the server and let Compose build from the Git URL:

```yaml
services:
  bot:
    build: https://github.com/darkost12/docker-monitor-tg.git#main
    container_name: docker-tg-bot
    restart: unless-stopped
    env_file: .env
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    cap_drop:
      - ALL
    security_opt:
      - no-new-privileges:true
    mem_limit: 128m
```

Update to the latest `main` with:

```bash
docker compose up -d --build
```

You deploy whatever is on `main` at that moment. To pin a version, use a tag or commit instead of `#main` (for example `#v1.0`).

## Security notes

- **Access to the Docker socket is powerful.** On a Linux host it is effectively root access to the machine. This bot only reads (it runs `docker ps` and `docker logs`), and it only answers the configured user, but treat the bot token as a secret. Consider putting a read-only Docker socket proxy in front of the socket.
- Keep `.env` out of version control (it is in `.gitignore` and `.dockerignore`).
- **Do not paste raw logs publicly.** Network errors from the Telegram library can contain the full API URL, and that URL includes your bot token. If a token ever leaks, revoke it with `/revoke` in @BotFather.
- Container logs often contain secrets as well, and the bot sends them to Telegram. Only run it on machines where that is acceptable.
- Authorization is by numeric user ID, not by username.

## Troubleshooting

| Symptom | What to check |
|---|---|
| `i/o timeout` when starting, requests to `api.telegram.org` fail | Telegram is not reachable from this machine. Set `HTTPS_PROXY`, or run the bot on a machine with direct access. Testing with `curl -v https://api.telegram.org` shows whether the network is the problem. |
| Bot answers with "Docker not running" | The Docker daemon is stopped, or the socket is not mounted into the container. |
| `permission denied` on the Docker socket | The user running the bot has no access to the socket (on Linux, add it to the `docker` group, or run the container as root). |
| "Container not found" | Check the name or ID with `/ps`. |
| Logs still contain `[96m`, `[0m` and the like | Make sure the running bot is built from the latest code. Rebuild with `docker compose up -d --build` on the machine where the bot actually runs. |
| `409 Conflict: terminated by other getUpdates request` | Two instances of the bot use the same token. Stop one of them. |
| `client version ... is too new` | The `docker` CLI in the image is newer than the daemon. Set `DOCKER_API_VERSION` to the daemon's API version in the container environment. |

## Project structure

```
cmd/bot/              entry point: loads config, wires components, handles signals
internal/config/      loading settings from .env and the environment
internal/docker/      thin wrapper over the docker CLI (timeout, name validation)
internal/textutil/    trimming to Telegram limits, stripping ANSI codes
internal/bot/         command logic (Service) and Telegram connection (Run)
```

The command logic in `internal/bot` does not depend on Telegram or on a real Docker: it talks to a small `Docker` interface, so it is tested with a fake.

## Development

```bash
go vet ./...
go test -race -cover ./...
gofmt -l .
```

Tests use fakes instead of a real Docker daemon or Telegram, so they run anywhere, with no network.
