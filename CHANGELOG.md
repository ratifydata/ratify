# Changelog

All notable changes to Ratify are recorded here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Ratify uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Until v1.0, breaking changes may happen between minor versions.
After v1.0, we commit to not breaking the API within a major version.

---

## [Unreleased]

### Added
- Initial project structure and repository setup
- Go module initialised (`github.com/ratifydata/ratify`)
- React + TypeScript + Vite frontend scaffold
- GitHub Actions CI pipeline (Backend, Frontend, Security, PR Title Format)
- Docker Compose local development environment with PostgreSQL 16
- Two-stage production Dockerfile, final image under 50MB
- Air live-reload development Dockerfile
- Environment variable loading via Viper (`internal/config`)
- Database migration runner via golang-migrate (`internal/db`)
- `GET /health` endpoint - returns `{"status":"ok","database":"ok","version":"0.1.0"}`
- HTTP server with graceful shutdown on SIGINT/SIGTERM
- Chi router with Logger and Recoverer middleware
- pgx/v5 connection pool
- pgcrypto extension enabled via migration 000001 (required for UUID generation)
- Structured JSON logging via `log/slog`
- golangci-lint v2 configuration
- Prettier + ESLint frontend code quality tooling
- `.gitattributes` enforcing Unix line endings

---

## How this file works

Every pull request that changes behaviour - new features, bug fixes,
deprecations, removals - adds an entry to the `[Unreleased]` section
above under the appropriate heading.

When we cut a release, the `[Unreleased]` section becomes the new
version entry (e.g. `[0.2.0] - 2026-07-01`) and a fresh `[Unreleased]`
section is opened above it.

**Heading types used in this file:**

- `Added` - new features or capabilities
- `Changed` - changes to existing behaviour
- `Deprecated` - features that will be removed in a future version
- `Removed` - features removed in this version
- `Fixed` - bug fixes
- `Security` - security fixes (always include the CVE or advisory link if one exists)

Not every commit needs a changelog entry. Dependency updates, refactors that do not change behaviour, and internal tooling changes do not need one. Features, fixes, and anything that changes how the tool behaves do.