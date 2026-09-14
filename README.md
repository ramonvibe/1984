# 1984

[English](README.md) · [Português brasileiro](README.pt-BR.md)

A lightweight, self-hostable issue and project tracker for small software teams.

> Everything a small software team needs. Nothing it doesn't.

1984 is a modular Go monolith with PostgreSQL and server-rendered HTML. No Node.js, frontend build pipeline, Redis, or separate worker is required to run it.

This is an early functional version. The interface and Go module still use the working name **Trackline**.

## Features

- Workspace onboarding; the first user becomes an admin. Admins can add members.
- Projects with readable issue identifiers such as `PLAT-142`.
- Bugs, features, improvements and tasks; fixed statuses and priorities.
- Issue properties edited with HTMX, labels, Markdown comments and activity history.
- Drag-and-drop board; cards display the assignee's first name beside their avatar.
- Sprints with start/end dates, inclusive duration and board filtering. Assign issues through their Sprint property.
- One active timer per user, manual time entries and estimated-versus-spent reports.
- Dashboard and time reports by member, project, issue, type and release.
- Calendar for issue dates and release targets; PostgreSQL-based search.
- Releases with editable, deterministic changelog drafts from completed issues.
- Optional GitHub App integration for linked commits/PRs and publishing GitHub Releases.

## Run locally

Requirements: Go 1.25 or later, PostgreSQL (tested with version 18), Git, Bash and optionally Make.

```bash
git clone https://github.com/ramonvibe/1984.git
cd 1984
```

Create a dedicated database and non-superuser role using a PostgreSQL administrator account. Adapt the connection options to your installation:

```bash
psql -h 127.0.0.1 -U postgres -d postgres
```

Inside `psql`:

```sql
CREATE ROLE trackline LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE;
\password trackline
CREATE DATABASE trackline OWNER trackline;
REVOKE ALL ON DATABASE trackline FROM PUBLIC;
\q
```

If another project already uses PostgreSQL, reuse the server but keep this database and role separate. Do not overwrite an existing database with the same name.

Create `.env` in the repository root, replacing the example password with the one you chose. URL-encode special characters in the connection URL:

```dotenv
DATABASE_URL='postgres://trackline:YOUR_PASSWORD@127.0.0.1:5432/trackline?sslmode=disable'
ADDR=':8080'
BASE_URL='http://localhost:8080'
```

```bash
chmod 600 .env
make dev
# Without Make: bash scripts/dev.sh
```

Open [localhost:8080](http://localhost:8080), create your workspace and admin account, then create a project. The server applies pending migrations automatically. Restart `make dev` after code changes; there is no hot reload.

`.env` is ignored by Git. The development script sources it as Bash, so only use a trusted file. `sslmode=disable` is for local development, not untrusted networks.

## Configuration and hosting

The application reads environment variables. Only the development script loads `.env` automatically.

| Variable | Purpose |
| --- | --- |
| `DATABASE_URL` | PostgreSQL connection URL; set explicitly for your installation. |
| `ADDR` | HTTP listen address; default `:8080`. |
| `BASE_URL` | Public application URL; default `http://localhost:8080`. |
| `SECURE_COOKIES` | Defaults to true when `BASE_URL` starts with `https://`. |
| `GITHUB_APP_ID` | Optional GitHub App ID. |
| `GITHUB_APP_SLUG` | Optional App slug used by the installation link. |
| `GITHUB_PRIVATE_KEY` | App RSA private key in PEM format; literal `\n` sequences are supported. |
| `GITHUB_WEBHOOK_SECRET` | App webhook secret, at least 32 bytes. |

Without GitHub configuration, the tracker works independently. Once the App is configured and installed, connect its installation ID and repository in project settings. The webhook endpoint is `POST /webhooks/github`. Keep keys and secrets outside Git.

Build a single executable with embedded templates, assets and migrations:

```bash
go build -o /tmp/1984-server ./cmd/server
# With DATABASE_URL and other variables exported:
/tmp/1984-server -migrate
/tmp/1984-server
```

For hosting, use HTTPS through a reverse proxy, set the public `BASE_URL`, protect database access and keep PostgreSQL backups. Complete onboarding before exposing an empty installation publicly. `GET /healthz` checks database connectivity. Review security and recovery procedures before production use.

## Development

```bash
go test ./...
```

The optional sprint database integration test runs when `TEST_DATABASE_URL` is set. Use a test database whose role can create schemas; it creates and removes an isolated temporary schema.

Templates and SQL query bindings are committed, so generators are not required for a normal build. After editing `.templ` files or SQL queries, regenerate the corresponding Go files:

```bash
go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate
```

Keep HTTP handlers in `internal/server`, business rules in the domain packages, SQL in `db/queries`, schema changes in new `db/migrations` files and presentation in `web`. Do not edit generated `*_templ.go` or query bindings manually, or change migrations already applied to a database.

Contributions should stay small, include relevant tests and avoid unnecessary dependencies. Never commit `.env`, credentials or database dumps.

## Current limitations

- The UI is English-only; Portuguese UI translation is pending. Both README languages are available.
- Sprint assignment is manual. Automatically assigning issues from Todo onward to the current sprint is pending.
- Dockerfile and Docker Compose are not included yet.
- Project membership is organizational: all workspace members can access every project. There is no per-project access control.
- No license file has been chosen yet. Do not assume open-source redistribution rights until a license is added.
