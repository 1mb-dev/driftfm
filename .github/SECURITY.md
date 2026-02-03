# Security Policy

## Reporting a Vulnerability

If you find a security issue, please open a [private security advisory](https://github.com/1mb-dev/driftfm/security/advisories/new) instead of a public issue.

Include:
- Description of the vulnerability
- Steps to reproduce
- Impact assessment

## Scope

Drift FM is designed to run on a private network or behind a reverse proxy. The threat model assumes a trusted operator. Shell scripts (import, normalize) are run locally by the operator and are not exposed to network input.

## Supported Versions

Only the latest release on `main` is supported.
