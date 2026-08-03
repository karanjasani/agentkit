# Security Policy

## Supported Versions

AgentKit is pre-1.0. Security fixes are applied to the latest released minor
version. Once 1.0 is reached, this table will list the supported release lines.

| Version | Supported |
| ------- | --------- |
| 0.1.x   | ✅        |
| < 0.1   | ❌        |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, report them privately using GitHub's
[private vulnerability reporting](https://github.com/karanjasani/agentkit/security/advisories/new)
(the "Report a vulnerability" button under the repository's **Security** tab).

Please include:

- A description of the vulnerability and its impact.
- Steps to reproduce, ideally with a minimal proof of concept.
- Affected version(s) and platform.
- Any suggested remediation, if you have one.

## What to expect

- **Acknowledgement** within 3 business days.
- **Initial assessment** within 7 business days.
- We will keep you informed of progress and coordinate a disclosure timeline with
  you. We aim to release a fix within 90 days of a confirmed report.
- With your permission, we will credit you in the release notes.

## Scope

AgentKit is a read-only, offline CLI. It does not modify source, git, or logs, and
makes no network calls during analysis. Nonetheless, we take the following classes of
issue seriously:

- Path traversal or reads outside the target module root.
- Denial of service via crafted input (e.g. pathological source files).
- Any behavior that mutates the filesystem, git state, or makes network calls.
- Supply-chain integrity of released binaries.

Thank you for helping keep AgentKit and its users safe.
