# Security Policy

## Supported Versions

Ratify is currently in pre-release development. Until a stable
v1.0 is tagged, only the latest commit on the `main` branch
receives security fixes. We do not backport fixes to earlier
commits.

| Version | Supported |
|---|---|
| `main` (pre-release) | Yes |
| Any specific pre-release tag | No |

Once we reach v1.0, this table will be updated to reflect a
proper long-term support policy.

---

## Reporting a Vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

A public issue gives anyone watching the repository, including
people with bad intentions an advance notice before a fix is
available. Even if the vulnerability seems minor, please treat
it as private until we have had a chance to assess and patch it.

**How to report:**

Send an email to **[This Mail](mailto:lewiskunta5@gmail.com)** with the subject
line: `[SECURITY] Brief description of the issue`

We check this address regularly. You will receive an
acknowledgement within 48 hours. If you have not heard back
within 72 hours, follow up with us, your email may have gone to spam.

---

## What to Include in Your Report

A good report gives us enough information to reproduce the
issue and assess the severity without needing multiple rounds
of back-and-forth. Include:

- A clear description of the vulnerability
- The component or file where you found it (e.g. the connection
  credential storage, the response token generation, the API
  authentication middleware)
- Steps to reproduce it - the more specific the better
- What an attacker could do with it if exploited
- The version or commit hash where you found it
- Your suggested fix if you have one

You do not need to have a working exploit to report something.
If you have found something that looks wrong, tell us.

---

## What Happens After You Report

**Within 48 hours:** We acknowledge your report and confirm
we have received it.

**Within 7 days:** We assess the severity, determine whether
it is a genuine vulnerability in our code or in a dependency,
and tell you our conclusion. If we cannot reproduce it, we
will ask follow-up questions.

**Within 30 days (for confirmed vulnerabilities):** We aim
to have a fix developed, tested, and ready to release. For
critical vulnerabilities affecting credential exposure or
authentication bypass, we move faster.

**When we release the fix:** We will publish a GitHub Security
Advisory describing the vulnerability, the affected versions,
and the fix. We will credit you by name unless you have asked
to remain anonymous.

If for any reason we need more time than 30 days, we will
tell you why and give you a revised timeline. We will not
leave you without updates.

---

## Scope

### What we consider in scope

These are the things we care most about:

**Credential exposure**
Ratify stores encrypted database credentials. Anything that
could expose these credentials in plaintext; whether through
the API, the logs, error messages, or a vulnerability in the
encryption implementation - is a critical issue.

**Authentication bypass**
Any way to access the API or perform actions without a valid
API key or session token.

**Consumer response token weaknesses**
The response tokens sent to consumers in email links must be
single-use, time-limited, and unpredictable. Anything that
weakens these properties is in scope.

**Audit trail integrity**
The audit trail is designed to be append-only. Any way to
modify or delete audit events is a serious issue.

**SQL injection**
We use parameterised queries via sqlc. If you find a code
path that constructs queries with user input directly, that
is a serious finding.

**Privilege escalation**
Any way for a user in one organisation to access data from
another organisation, or for a lower-privileged user to
perform actions reserved for higher-privileged users.

**Remote code execution**
Any path that allows arbitrary code execution on the server.

### What we consider out of scope

These will not be treated as vulnerabilities:

- Vulnerabilities in software we do not control (the underlying
  Linux OS, Docker itself, PostgreSQL), report those to the
  respective projects
- Attacks that require physical access to the server
- Social engineering attacks against team members
- Theoretical vulnerabilities with no practical attack path
- Issues that only affect self-hosted deployments where the
  attacker already has shell access to the server (at that
  point the attacker has won regardless of what Ratify does)
- Missing security headers on the web UI in development mode
- Rate limiting on endpoints that do not handle sensitive data
- Self-XSS (where the attacker must trick themselves)

If you are unsure whether something is in scope, report it
anyway. The worst outcome is that we tell you it is out of
scope. We would rather hear about something that turns out
to be nothing than miss something real.

---

## Safe Harbour

We will not take legal action against you for finding and
reporting a vulnerability in good faith. We ask that you:

- Give us a reasonable amount of time to fix it before
  disclosing it publicly (30 days is our target, and we
  will keep you updated on progress)
- Make a genuine effort to avoid accessing, modifying, or
  deleting data that does not belong to you during your research
- Do not run automated scanners against production systems
  without prior agreement - test against your own instance

We consider responsible disclosure a service to the community
and we treat reporters accordingly.

---

## A Note on Dependencies

Ratify depends on third-party Go packages. If you find a
vulnerability in one of our dependencies, please also report
it to that project's maintainers directly. Let us know as well
so we can prioritise updating the affected package.

You can check for known vulnerabilities in our dependencies
yourself by running:

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

We run this check automatically in our CI pipeline on every
pull request.

---

## Credit and Disclosure

We maintain a record of security issues and the people who
reported them. When we publish a fix, we will credit your
name (or handle, or "anonymous" - your choice) in the
GitHub Security Advisory.

We do not currently offer a bug bounty programme. If that
changes, we will update this document.

---

*Last updated: May 2026*
