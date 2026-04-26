# proletarka transport

Small delivery service for outbound events from `proletarka`.

It accepts `POST /events`, validates a shared secret from `x-outbound-events-secret`, and dispatches supported events to delivery channels.

Current scope:
- one inbound event: `comment.created`
- delivery order: `Telegram -> Email`
- success if at least one channel delivers
- failure if all configured channels fail

This service is intentionally small:
- no database
- no queue
- no background workers
- no delivery history
- no rule engine
- no direct access to `proletarka` database

## Requirements

- Go 1.22+

## Configuration

Copy [.env.example](/home/tuor4eg/pets/proletarka_transport/.env.example) and set the needed variables.

### Required

- `INBOUND_EVENTS_SECRET`

### Optional with defaults

- `BIND_ADDR`
  Default: `0.0.0.0`
- `PORT`
  Default: `8080`

### Telegram channel

Telegram is enabled only when both variables are set:

- `TELEGRAM_BOT_TOKEN`
- `TELEGRAM_CHAT_IDS`

`TELEGRAM_CHAT_IDS` is a comma-separated list with one or more Telegram chat ids, for example:

```text
TELEGRAM_CHAT_IDS=12345,67890,99999
```

### Email channel

Email is enabled only when all variables are set:

- `SMTP_HOST`
- `SMTP_PORT`
- `SMTP_USER`
- `SMTP_PASSWORD`
- `EMAIL_FROM`
- `EMAIL_TO`

`EMAIL_TO` is a comma-separated list with one or more email recipients, for example:

```text
EMAIL_TO=admin1@example.com,admin2@example.com
```

The service will not start if:
- `INBOUND_EVENTS_SECRET` is missing
- both Telegram and Email are disabled
- a channel is configured only partially

## Run

Build:

```bash
go build -o transport ./cmd/transport
```

Or with `Makefile`:

```bash
make build
```

Run:

```bash
BIND_ADDR=0.0.0.0 INBOUND_EVENTS_SECRET=change-me PORT=8080 ./transport
```

For local shell-based runs you can export variables first or use your preferred env loader.

Useful `Makefile` commands:
- `make build`
- `make run`
- `make test`
- `make tidy`
- `make fmt`
- `make clean`
- `make deploy`

## Production run without Docker

Recommended approach for this service:
- keep the repository on the server
- keep env in a separate file
- run the service with `systemd`
- deploy with `git pull` and local build on the server

Example unit file:
- [deploy/systemd/proletarka-transport.service](/home/tuor4eg/pets/proletarka_transport/deploy/systemd/proletarka-transport.service)

Example server layout:

```text
/srv/proletarka-transport/repo
/opt/proletarka_transport/transport
/opt/proletarka_transport/.env
```

Example env file at `/opt/proletarka_transport/.env`:

```bash
BIND_ADDR=0.0.0.0
PORT=8080
INBOUND_EVENTS_SECRET=change-me
TELEGRAM_BOT_TOKEN=
TELEGRAM_CHAT_IDS=
SMTP_HOST=
SMTP_PORT=
SMTP_USER=
SMTP_PASSWORD=
EMAIL_FROM=
EMAIL_TO=
```

Basic `systemd` flow:

```bash
sudo cp deploy/systemd/proletarka-transport.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable proletarka-transport
sudo systemctl start proletarka-transport
sudo systemctl status proletarka-transport
```

After initial setup, you can deploy updated versions on the server from the repo directory with:

```bash
make deploy
```

Default `make deploy` behavior:
- runs `git pull --ff-only origin main`
- builds `./cmd/transport`
- installs the binary to `/opt/proletarka_transport/transport`
- restarts `proletarka-transport`
- prints service status

You can override defaults if needed:

```bash
make deploy DEPLOY_BRANCH=master DEPLOY_DIR=/opt/proletarka_transport SYSTEMD_SERVICE=proletarka-transport
```

For a Docker-on-host setup, you can bind the service only to the Docker bridge IP instead of exposing it on all interfaces:

```bash
BIND_ADDR=172.17.0.1
PORT=8080
```

With this setup, containers can use `http://host.docker.internal:8080/events`, while the transport service is not exposed on public interfaces.

## HTTP API

### `POST /events`

Headers:

```text
Content-Type: application/json
x-outbound-events-secret: <shared-secret>
```

Request body:

```json
{
  "event": "comment.created",
  "occurredAt": "2026-04-09T10:00:00Z",
  "severity": "normal",
  "resource": {
    "kind": "comment",
    "id": "comment_123"
  },
  "payload": {
    "comment": {
      "text": "Looks good",
      "authorName": "Alice"
    },
    "target": {
      "type": "post",
      "title": "Release Notes"
    },
    "urls": {
      "public": "https://example.com/posts/release-notes",
      "admin": "https://admin.example.com/comments/comment_123"
    }
  }
}
```

Validation rules:
- `event` is required
- `occurredAt` must be valid RFC3339
- `severity` must be `low`, `normal`, or `high`
- `resource.kind` is required
- `resource.id` is required
- `payload` is required

Current supported event:
- `comment.created`

For `comment.created`, the service currently expects comment text in one of these places:
- `payload.comment.text`
- `payload.commentText`

Optional fields used to build delivery messages:
- `payload.comment.authorName`
- `payload.comment.author`
- `payload.target.type`
- `payload.target.title`
- `payload.urls.public`
- `payload.urls.admin`
- `payload.publicUrl`
- `payload.adminUrl`

Responses:
- `200 OK` when the event is processed and at least one channel delivers
- `400 Bad Request` for invalid JSON, invalid event payload, or unsupported event
- `401 Unauthorized` when the shared secret is invalid
- `405 Method Not Allowed` for methods other than `POST`

## Delivery behavior

For `comment.created`:

1. Try Telegram.
2. If Telegram fails, try Email.
3. If either one succeeds, the whole request is treated as successful.
4. If both fail, the service returns an error and writes readable logs.

Telegram messages are currently sent as plain text without markup, so user-generated content is delivered safely without HTML parsing issues.
If multiple Telegram chat ids are configured, the Telegram channel is treated as successful when delivery succeeds for at least one recipient.
If multiple email recipients are configured, the Email channel is treated as successful when delivery succeeds for at least one recipient.

## Telegram commands

Current commands:
- `/start` shows the button menu
- `/ping` replies with `pong`

Current buttons:
- `Ping` replies with `pong`

Command rules:
- commands are available only for Telegram ids listed in `TELEGRAM_CHAT_IDS`
- the service checks access before running the command handler
- if `TELEGRAM_CHAT_IDS` is empty, command handling is disabled

## Logs

The service writes simple structured logs for:
- startup
- rejected requests
- accepted event processing
- channel delivery attempts
- channel delivery success
- channel delivery failure

## Project structure

- [cmd/transport/main.go](/home/tuor4eg/pets/proletarka_transport/cmd/transport/main.go) application entrypoint
- [internal/config/config.go](/home/tuor4eg/pets/proletarka_transport/internal/config/config.go) env loading and validation
- [internal/http/events_handler.go](/home/tuor4eg/pets/proletarka_transport/internal/http/events_handler.go) `POST /events`
- [internal/domain/event.go](/home/tuor4eg/pets/proletarka_transport/internal/domain/event.go) inbound event contract
- [internal/events/dispatcher.go](/home/tuor4eg/pets/proletarka_transport/internal/events/dispatcher.go) dispatch by event name
- [internal/events/comment_created.go](/home/tuor4eg/pets/proletarka_transport/internal/events/comment_created.go) `comment.created` delivery flow
- [internal/channels/channel.go](/home/tuor4eg/pets/proletarka_transport/internal/channels/channel.go) shared channel interface
- [internal/channels/telegram.go](/home/tuor4eg/pets/proletarka_transport/internal/channels/telegram.go) Telegram delivery
- [internal/channels/email.go](/home/tuor4eg/pets/proletarka_transport/internal/channels/email.go) Email delivery
