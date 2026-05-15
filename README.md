<div align="center">

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/assets/ratifydata-dark-full.png">
  <source media="(prefers-color-scheme: light)" srcset="docs/assets/ratifydata-light-full.png">
  <img alt="Ratify Logo" src="docs/assets/ratifydata-light-full.png" width="450">
</picture>

# Ratify

**The missing workflow layer between data producers and consumers.**

[![CI](https://github.com/ratifydata/ratify/actions/workflows/ci.yml/badge.svg)](https://github.com/ratifydata/ratify/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](https://github.com/ratifydata/ratify/blob/main/LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8.svg?logo=go&logoColor=white)](https://golang.org)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16+-336791.svg?logo=postgresql&logoColor=white)](https://postgresql.org)
[![Discord](https://img.shields.io/discord/1504808765768073309?label=discord&logo=discord&logoColor=white&color=5865F2)](https://discord.gg/RVU3cdRs9z)
[![Stars](https://img.shields.io/github/stars/ratifydata/ratify?style=social)](https://github.com/ratifydata/ratify)
[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE_OF_CONDUCT.md)

[Documentation](https://github.com/ratifydata/ratify/tree/main/docs) &nbsp;·&nbsp;
[Discord Community](https://discord.gg/RVU3cdRs9z) &nbsp;·&nbsp;
[Report a Bug](https://github.com/ratifydata/ratify/issues) &nbsp;·&nbsp;
[LinkedIn](https://linkedin.com/in/lewisinjai)

</div>

---

> **Note**
> Ratify is under active development and currently in Phase 1 (Core Loop).
> We are building in public, every decision is shared openly.
> Expect rough edges. We welcome honest feedback.

---

A schema change ships on Friday. Nobody tells the data team. The pipeline breaks on Monday. The executive dashboard goes stale. Everyone loses a week they didn't have.

**Ratify** is an open-source data contract workflow engine. It gives data teams a structured, auditable process for proposing, negotiating, and approving schema changes - *before* those changes cause downstream failures.

---

## The problem in one sentence

Data teams have no GitHub PR equivalent for schema changes. Ratify builds that.

---

## How it works

```
Producer proposes a change
        ↓
System classifies it (breaking or additive)
        ↓
Consumer teams are notified (email or Slack)
        ↓
Consumers accept, reject, or request a migration period
        ↓
Proposal approved → contract version activated → audit trail updated
        ↓
Breach detection runs hourly - catches anything that slips through
```

No dashboards to log into. No accounts required for consumers to respond.
The whole flow lives where your team already works.

---

## Features

### Contract Authoring
Define what a dataset promises; column names, types, nullability, freshness SLA,
and value constraints. Contracts are validated against your live database schema
on creation and continuously monitored for drift.

### Change Proposal System
Before changing a schema, producers raise a formal proposal. Every change is
automatically classified as `breaking` or `additive`. Breaking changes require
explicit consumer approval before they can proceed.

### Consumer Notifications + Response
Consumers receive a notification with full context and a secure one-click response
link. **No account required.** Responses (accept / reject / request migration period)
are recorded immediately and visible to the producer in real time.

### Breach Detection
A scheduled job runs every hour and compares your live database schema against
active contract snapshots. When reality diverges from what was agreed, a column
is dropped, a type changes, a table disappears and the right people are alerted
immediately.

### Immutable Audit Trail
Every event is written to an append-only log. Who raised a proposal, who approved
it, when a breach was detected, what the schema looked like at each version.
Answering *"who agreed to this change?"* takes seconds, not hours.

### Slack Integration *(Phase 2)*
Proposals arrive in Slack as interactive messages. Consumers accept or reject with
one click without leaving the channel.

### GitHub / GitLab PR Integration *(Phase 2)*
Ratify inspects incoming PRs for migration files. If a migration touches a column
covered by an active contract, the PR is flagged before merging.

---

## Quickstart

### Requirements
- Docker and Docker Compose
- A PostgreSQL database (read-only credentials sufficient)

### Run with Docker Compose

```bash
git clone https://github.com/ratifydata/ratify.git
cd ratify
cp .env.example .env
docker compose up
```

The UI is available at `http://localhost:8080`. That's it.

### Connect your first database

```bash
ratify connect add \
  --name "production-db" \
  --host db.example.com \
  --port 5432 \
  --database myapp \
  --username ratify_readonly
```

Ratify only requires `SELECT` on `information_schema` and `pg_catalog`.
It **never writes** to your database.

### Create your first contract

```bash
ratify contract create \
  --connection production-db \
  --schema public \
  --table orders
```

Ratify pre-populates the contract from your live schema. Review, describe, assign a producer team, and activate.

### Raise a change proposal

```bash
ratify proposal create \
  --contract orders \
  --title "Rename order_status to status" \
  --deadline 2026-06-01
```

Consumer teams are notified by email immediately.

---

## CLI Reference

```
ratify connect add          Register a PostgreSQL connection
ratify connect test         Test a saved connection
ratify connect list         List all connections

ratify contract create      Create a new contract interactively
ratify contract list        List all contracts
ratify contract show        Show contract detail and current status
ratify contract activate    Activate a draft contract
ratify contract deprecate   Deprecate an active contract

ratify proposal create      Create a change proposal
ratify proposal list        List open proposals
ratify proposal show        Show proposal detail and responses
ratify proposal withdraw    Withdraw an open proposal

ratify breach list          List open breaches
ratify breach acknowledge   Acknowledge a breach

ratify audit log            View recent audit trail events
```

All commands support `--output json` for scripting and CI pipelines.

---

## Tech Stack

| Layer | Technology | Why |
|---|---|---|
| Backend | Go 1.22+ | Single binary, no runtime, best PostgreSQL driver |
| CLI | Cobra + Viper | Industry standard, used by kubectl and GitHub CLI |
| API | Chi router | Stdlib-compatible, no magic |
| Database | PostgreSQL + sqlc | Type-safe queries, no ORM, explicit SQL |
| Migrations | golang-migrate | Plain SQL files, version controlled |
| Frontend | React + TypeScript + Vite | Typed, fast, large contributor pool |
| Auth | API Keys + JWT | Lowest friction for self-hosted OSS |
| Scheduler | robfig/cron | In-process, no separate worker needed |
| Email | go-mail | Modern SMTP, TLS, HTML templates |
| Containers | Docker Compose | One command to running state |

---

## Architecture

```
┌─────────────────────────────────────────────┐
│              Client Layer                   │
│         CLI Binary         Web UI           │
└──────────────┬─────────────────┬────────────┘
               │                 │
               ▼                 ▼
┌─────────────────────────────────────────────┐
│            REST API  /api/v1/               │
│    Auth · Contracts · Proposals · Breaches  │
│    Audit · Notifications · Scheduler        │
└───────────────────┬─────────────────────────┘
                    │
        ┌───────────┴───────────┐
        ▼                       ▼
┌──────────────┐      ┌──────────────────────┐
│  Ratify DB   │      │   Your PostgreSQL DB  │
│  (metadata)  │      │   READ-ONLY           │
│              │      │   information_schema  │
└──────────────┘      └──────────────────────┘
```

Ratify **never writes** to your database. It only reads schema metadata.

---

## Configuration

All configuration is via environment variables. Copy `.env.example` to `.env`:

```env
# Application
PORT=8080
ENVIRONMENT=production

# Ratify's own metadata database
DATABASE_URL=postgresql://ratify:password@postgres:5432/ratify

# Security — generate with: openssl rand -hex 32
ENCRYPTION_KEY=your-32-byte-hex-key-here
JWT_SECRET=your-jwt-secret-here

# Email (required for notifications)
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USERNAME=ratify@example.com
SMTP_PASSWORD=your-smtp-password
SMTP_FROM=ratify@example.com

# Breach detection interval (default: 1h)
BREACH_DETECTION_INTERVAL=1h
```

---

## Security

- Database credentials encrypted at rest with **AES-256-GCM**
- API keys hashed with **bcrypt**,  never stored in plaintext
- Consumer response tokens are **single-use, time-limited, SHA-256 hashed**
- Audit trail is **append-only**, no record can be modified or deleted
- **No telemetry** sent to external services without explicit opt-in

To report a security vulnerability, please email [This email](mailto:lewiskunta5@gmail.com)
rather than opening a public issue.

---

## Deployment

### Docker Compose (recommended)

```bash
docker compose up -d
```

### Single Binary

Download from [Releases](https://github.com/ratifydata/ratify/releases):

```bash
# Linux amd64
curl -L https://github.com/ratifydata/ratify/releases/latest/download/ratify-linux-amd64 \
  -o ratify && chmod +x ratify
./ratify server start

# macOS arm64 (Apple Silicon)
curl -L https://github.com/ratifydata/ratify/releases/latest/download/ratify-darwin-arm64 \
  -o ratify && chmod +x ratify
./ratify server start

# Windows amd64 (PowerShell)
Invoke-WebRequest -Uri "https://github.com/ratifydata/ratify/releases/latest/download/ratify-windows-amd64.exe" -OutFile "ratify.exe"
.\ratify.exe server start
```

### Health Check

```bash
curl http://localhost:8080/health
# {"status":"ok","database":"ok","scheduler":"ok","version":"0.1.0"}
```

---

## Roadmap

| Phase | Status | Description |
|---|---|---|
| Phase 0: Foundation | In progress | Codebase skeleton, auth, Docker |
| Phase 1: Core Loop | Planned | Contracts, proposals, breach detection, audit trail |
| Phase 2: Integrations | Planned | Slack, GitHub/GitLab PR hooks, Web UI |
| Phase 3: Intelligence | Planned | Auto-suggest contracts, drift detection |
| Phase 4: Breadth | Planned | SQL Server, RBAC, outbound webhooks |
| Phase 5: SaaS | Planned | Hosted version, SSO, billing |

Building in public. Every decision shared on [LinkedIn](https://linkedin.com/in/lewisinjai).

---

## Contributing

Contributions are welcome. The best contributions right now are honest feedback,
bug reports, and ideas from real data teams using real databases.

### Local setup

```bash
git clone https://github.com/ratifydata/ratify.git
cd ratify
docker compose up postgres -d   # start the metadata database
go run ./cmd/server             # start the backend
cd frontend && npm install && npm run dev  # start the UI
```

### Before submitting a PR

```bash
golangci-lint run ./...   # Go linting — must pass
go test ./...             # all tests must pass
npm run lint              # frontend linting — must pass
```

See [CONTRIBUTING.md](https://github.com/ratifydata/ratify/blob/main/CONTRIBUTING.md) for the full guide including
branch naming, commit message format, and the PR review process.

---

## What Ratify is not

| Tool type | Use instead |
|---|---|
| Data quality (row-level) | Great Expectations, Soda |
| Data observability / anomaly detection | Monte Carlo, Bigeye |
| Data catalog | Atlan, Collibra, DataHub |
| Kafka schema registry | Confluent Schema Registry |

Ratify does one thing: structured workflow for agreeing on schema changes
before they happen, with a reliable record of every agreement.

---

## Why we built this

In separate discussions with a CTO at a commercial bank and a Head of Digital Transformation at another, we identified a recurring organizational failure: upstream teams shipping schema changes without notifying data consumers.

The Head of Infrastructure at the first bank described the fallout, broken pipelines, stale reports, and constant conflict between engineering and data teams. Their temporary "solution" was to hire a DBA with a firm personality to enforce contracts manually.

That is not a solution. That is a missing infrastructure problem.

No existing tool addresses the workflow and negotiation layer required for data contracts. Ratify replaces manual "firmness" with a structured process, ensuring changes are approved before they hit production.

---

## License

[MIT License](LICENSE) - free to use, modify, and distribute.

---

<div align="center">

**[Documentation](https://github.com/ratifydata/ratify/tree/main/docs) · [Discord](https://discord.gg/RVU3cdRs9z) · [LinkedIn](https://linkedin.com/in/lewisinjai)**

<sub>Built in public · Honest about progress · PostgreSQL first</sub>

</div>
