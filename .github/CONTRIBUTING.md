# Contributing

Thanks for your interest in Drift FM.

## Bug Reports

Open an issue with:
- What you expected
- What happened
- Steps to reproduce
- Go version, OS

## Pull Requests

1. Fork and create a feature branch
2. Make your changes
3. Run `make check` (lint, vet, test)
4. Open a PR against `main`

Keep PRs focused — one concern per PR.

## Development

```bash
make db-init          # Set up database
make dev              # Hot reload (requires air)
make check            # Full quality gate
```

## Code Style

- Go: `gofmt` + `golangci-lint`
- JS: `eslint` with project config
- Shell: `set -e`, quote variables
