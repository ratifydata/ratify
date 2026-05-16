# Contributing to Ratify

First, thank you. Whether you are fixing a typo, reporting a bug,
or building a feature, you are helping data teams everywhere
stop finding out about schema changes after they've been made.

This document is the single source of truth for how to contribute.
Read it fully before opening an issue or a pull request.

---

## Table of Contents

- [Who should contribute](#who-should-contribute)
- [Ways to contribute](#ways-to-contribute)
- [Before you start coding](#before-you-start-coding)
- [Local development setup](#local-development-setup)
- [How we work](#how-we-work)
- [Branch naming](#branch-naming)
- [Writing good commits](#writing-good-commits)
- [Pull requests](#pull-requests)
- [Code standards](#code-standards)
- [Testing expectations](#testing-expectations)
- [Reporting security vulnerabilities](#reporting-security-vulnerabilities)
- [Getting help](#getting-help)

---

## Who should contribute

Anyone. We deliberately avoid the word "developer" here because
some of the most valuable contributions are not code:

- Data engineers who use the tool and have opinions about the workflow
- DBAs who can tell us what the CLI experience actually needs to feel like
- Technical writers who find the documentation confusing
- Anyone who has lived the problem of upstream schema changes breaking pipelines

If you have lived the problem, your perspective is worth more than
someone who has only read about it.

---

## Ways to contribute

**Report bugs.** If something is broken, open a GitHub issue.
Be specific: what you did, what you expected, what actually happened,
your operating system, Go version if relevant. A reproducible bug
report is one of the most valuable things you can give a project.

**Suggest features.** Open an issue with the `enhancement` label.
Describe the problem you are trying to solve, not just the solution
you have in mind. The problem statement is what helps us understand
whether there is a better approach we had not considered.

**Improve documentation.** If something in the docs is unclear,
incomplete, or wrong - fix it and open a PR. Documentation contributions
are always welcome and are a good first contribution for anyone
new to the codebase.

**Write code.** If you are new to the project, look for issues
labelled `good first issue`. These are scoped to be doable without
deep knowledge of the whole codebase. Issues labelled `help wanted`
are things we genuinely want community help with.

**Give honest feedback.** Use the tool on a real database and tell
us what is confusing or missing. At this stage of the project,
candid feedback from actual data teams matters more than almost
anything else.

---

## Before you start coding

**For small changes:** typos, documentation fixes, minor bug fixes:
go ahead and open a PR directly.

**For anything significant:** new features, API changes, architectural
decisions, database schema changes: open an issue first. Describe what
you want to build and why. Wait for a response before writing code.
This is not bureaucracy, it prevents you from spending time on
something that conflicts with the roadmap or duplicates work already
in progress.

**For security vulnerabilities:** do not open a public issue. Read
[SECURITY.md](SECURITY.md) for how to report responsibly.

---

## Local development setup

### What you need installed first

| Tool | Minimum version | Install | Verify |
|---|---|---|---|
| Go | 1.22 | [go.dev/dl](https://go.dev/dl) | `go version` |
| Node.js | 20 | [nodejs.org](https://nodejs.org) | `node --version` |
| Docker Desktop | Latest | [docker.com](https://www.docker.com/products/docker-desktop) | `docker --version` |
| Git | Any | Usually pre-installed | `git --version` |
| golangci-lint | Latest | [golangci-lint.run](https://golangci-lint.run/usage/install) | `golangci-lint --version` |
| sqlc | Latest | [docs.sqlc.dev](https://docs.sqlc.dev/en/latest/overview/install.html) | `sqlc version` |

Install them in order. Do not skip the version checks - the project
uses features from specific versions and older releases will produce
confusing errors.

### Getting the application running

```bash
# Clone the repository
git clone https://github.com/ratifydata/ratify.git
cd ratify

# Copy the environment file
# The defaults in .env.example work for local development
cp .env.example .env

# Start the application and its database
docker compose up
```

Wait about 30 seconds. When you see log output from both the `app`
and `postgres` containers without errors, the application is running.

Verify it in a separate terminal:

```bash
curl http://localhost:8080/health
```

Expected response:

```json
{"status":"ok","database":"ok","version":"0.1.0"}
```

If you see this, your environment is working and you are ready to write code.

### Frontend development

If you are working on the web UI:

```bash
cd frontend
npm install
npm run dev
```

The frontend is at `http://localhost:5173` and hot-reloads as you make changes.

### Stopping everything

```bash
# Stop containers, keep data
docker compose down

# Stop containers and remove all data (clean slate)
docker compose down -v
```

---

## How we work

A few things that are non-negotiable regardless of how small the change:

**No code goes directly to `main`.** Every change goes through a
pull request. The branch protection rule enforces this.

**The CI pipeline is the gatekeeper.** A pull request cannot be
merged until all CI checks pass; lint, tests, coverage, and security
scan. If CI is red, that is the highest priority. Everything else waits.

**Tests are not optional.** If you add behaviour, you add a test for it.
If you fix a bug, you add a test that would have caught it. PRs without
appropriate test coverage will not be merged.

**One PR, one thing.** A pull request should do one coherent thing.
Mixing a feature and a refactor in the same PR makes it harder to
review and harder to revert cleanly if something goes wrong.

---

## Branch naming

Every branch name follows this pattern: `type/short-description`

| Type | When to use |
|---|---|
| `feat/` | Adding new functionality |
| `fix/` | Fixing a bug |
| `docs/` | Documentation changes only |
| `setup/` | Infrastructure, tooling, configuration |
| `test/` | Adding or fixing tests without changing behaviour |
| `refactor/` | Restructuring code without changing behaviour |
| `perf/` | Performance improvements |

Rules:
- All lowercase, hyphens between words
- Short enough to read at a glance - `feat/consumer-response-tokens`
  not `feat/add-the-new-consumer-response-token-generation-endpoint`
- Always branch from the latest `main`:

```bash
git checkout main && git pull && git checkout -b feat/your-feature
```

---

## Writing good commits

We follow [Conventional Commits](https://www.conventionalcommits.org).
Every commit message starts with a type:

```
feat: add consumer response token generation
fix: prevent duplicate breach events on repeat detection
docs: clarify SSL configuration in README
test: add integration tests for proposal deadline processor
chore: update golangci-lint to v1.57
refactor: extract notification service from proposal handler
```

The summary line is 72 characters maximum, present tense, no full
stop at the end.

If the change needs explanation, add a blank line after the summary
and write a paragraph explaining **why** the change was made, not
what it does (the code shows that) but the reasoning behind it:

```
fix: prevent duplicate breach events on repeat detection

The breach detector was creating a new breach record on every
detection cycle when the original breach was still open. Over
a 24-hour period with hourly detection, one schema issue would
generate 24 separate breach alerts to the same people.

Added a check for existing open breaches on the same contract
and column before inserting a new record.

Closes #67
```

This history is permanent. Write for the person who will read this
commit message six months from now trying to understand why the
codebase looks the way it does.

---

## Pull requests

### Opening a PR

Push your branch and open a pull request against `main`.
Fill in the description using this structure:

```
## What does this PR do?
Brief description of the change.

## Why is this change needed?
The problem it solves or the feature it adds.
Link to the related issue if one exists.

## How to test it
Specific steps a reviewer can follow to verify this works.
Do not write "run the tests" — write the actual steps.

## Checklist
- [ ] Tests added or updated
- [ ] golangci-lint passes locally
- [ ] go test ./... passes locally
- [ ] Documentation updated where behaviour changed
- [ ] No hardcoded credentials or secrets
```

If your PR closes an issue, add `Closes #[number]` in the description.
GitHub will automatically close the issue when the PR is merged.

### The review process

A maintainer will review your PR within 48 hours. You will likely
receive feedback requesting changes, this is normal and is not a
rejection. Respond to feedback by pushing new commits to the same
branch. Do not close and re-open the PR.

Reviews look at:
- Does the code do what the PR description says it does?
- Are the tests meaningful, do they actually verify the behaviour?
- Does the approach fit the existing architecture?
- Is there a simpler way to achieve the same outcome?

### What blocks a merge

These will always block a merge, without exception:
- CI is failing (lint, tests, or security scan)
- Test coverage dropped below the minimum threshold
- A hardcoded credential or secret appears anywhere in the diff
- The PR does more than one unrelated thing
- A significant feature was built without a prior issue discussion

---

## Code standards

### Go

Ratify is written in idiomatic Go. If you are new to Go, read
[Effective Go](https://go.dev/doc/effective_go) before contributing
backend code, it is short and explains the reasoning behind
patterns you will encounter throughout the codebase.

**Errors are values. Handle them.**
Never discard an error with `_`. If an error genuinely cannot
happen in a specific context, add a comment explaining why.

```go
// Do this
result, err := doSomething()
if err != nil {
    return fmt.Errorf("doing something: %w", err)
}

// Never this
result, _ := doSomething()
```

**Wrap errors with context.**
When returning an error from a function, wrap it so the call
chain is visible without reading every intermediate function:

```go
return fmt.Errorf("creating proposal: %w", err)
```

**Context comes first.**
Any function that performs I/O - database, network, file - takes
`context.Context` as its first argument:

```go
func (s *ProposalService) Create(ctx context.Context, input CreateProposalInput) (*Proposal, error)
```

**No ORM. SQL goes in query files.**
Database queries are written as plain SQL in `internal/db/queries/`
and generated into typed Go code by sqlc. Do not write inline SQL
in Go files.

**Exported functions have comments.**
Every exported function, type, and constant has a godoc comment:

```go
// CreateProposal creates a new change proposal for the given contract.
// It classifies each proposed change and notifies registered consumers.
// Returns an error if the contract has an existing open proposal.
func (s *ProposalService) CreateProposal(ctx context.Context, input CreateProposalInput) (*Proposal, error) {
```

**No `panic` in production code paths.**
Return errors. Reserve `panic` for programmer errors you genuinely
did not expect, and not for runtime conditions.

Run before every push:

```bash
golangci-lint run ./...
go test ./...
```

### TypeScript / React

**No `any`.** Every variable and function parameter is typed.
If you are reaching for `any`, the type structure needs rethinking.

**Functional components only.** No class components.

**Server state goes through TanStack Query.** API calls do not
live in `useEffect` with manual loading state. They go through
`useQuery` and `useMutation`.

**Components do one thing.** If a component is doing data fetching
and rendering and form state management simultaneously, it is
three components. Split it.

Run before every push:

```bash
cd frontend && npm run lint && npm run type-check
```

### SQL and migrations

Every schema change is a numbered migration with both `.up.sql`
and `.down.sql` files:

```
migrations/
  000001_initial.up.sql
  000001_initial.down.sql
  000002_organizations.up.sql
  000002_organizations.down.sql
```

Rules:
- Never modify an existing migration - always create a new one
- Every `.down.sql` must cleanly reverse its `.up.sql`
- Column and table names use lowercase with underscores
- Every timestamp column uses `TIMESTAMPTZ`, not `TIMESTAMP`
- Foreign keys use actual `REFERENCES` constraints, not just
  matching column names

### General

- Configuration values come from environment variables. Nothing
  is hardcoded; not ports, not database names, not timeouts.
- No credentials or secrets appear in code, comments, or test files.
  If you need a placeholder credential in a test, use something
  obviously fake like `test-secret-not-real`.
- Delete commented-out code before opening a PR. If code is
  commented out, either remove it or explain why it exists -
  do not leave dead code for future readers to puzzle over.

---

## Testing expectations

**Unit tests** live in the same package as the code they test,
in a file ending with `_test.go`. They test individual functions
in isolation.

**Integration tests** use testcontainers-go to spin up a real
PostgreSQL database. They test the interaction between your code
and the database. They live in the same package, typically in a
file named `integration_test.go`.

**What makes a good test:**

- It tests behaviour, not implementation. If you rename an internal
  function, tests should not break.
- It is readable as documentation. Someone unfamiliar with the code
  should understand what the test verifies just by reading it.
- It is deterministic. It passes or fails the same way every time
  with no dependence on external state, clock time, or randomness.
- When it fails, the error message tells you exactly what went wrong.

**Coverage:**
The CI pipeline enforces a minimum test coverage threshold.
New code that brings coverage below this threshold blocks the merge.
Write tests as you write code, retrofitting tests onto a large
body of untested code is painful for everyone.

---

## Reporting security vulnerabilities

If you find a security vulnerability, do not open a public GitHub issue.
A public issue gives attackers advance notice before a fix is available.

Email us directly at the address in [SECURITY.md](SECURITY.md). Include:
- A description of the vulnerability
- Steps to reproduce it
- The potential impact you see
- Your suggested fix if you have one

We will acknowledge your report within 48 hours and keep you updated
as we work on a fix. We will credit you in the release notes unless
you prefer otherwise.

---

## Getting help

**Something is not working locally?**
Open a [GitHub issue](https://github.com/ratifydata/ratify/issues)
with the label `question`. Include the output of `docker compose ps`
and any error messages from the logs.

**Not sure if your idea fits the project?**
Start a [GitHub Discussion](https://github.com/ratifydata/ratify/discussions)
before investing time in code.

**Want to talk to the community?**
Join the [Discord server](https://discord.gg/RVU3cdRs9z).

---

Ratify is being built in public, one honest step at a time.
We are glad you are here.
