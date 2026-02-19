# Contributing to SepaQX

Thanks for your interest in contributing to **SepaQX** 🎉  
This project aims to be a small, reliable, production‑ready service for generating SEPA EPC QR codes, so we keep the contribution process strict but simple.

---

## Scope of Contributions

We welcome:

- Bug fixes
- Security improvements
- Performance optimizations
- Documentation improvements
- Test extensions
- Small, well‑scoped features that fit the project’s goals

Before starting large changes, please open an issue to discuss the idea.

---

## Development Setup

### Requirements

- Go (latest stable version recommended)
- Bash (for test scripts)
- Standard Unix tools (`curl`, `jq`, `awk`, etc.)

Clone the repository:

```bash
git clone https://github.com/safe-cap/sepaqx.git
cd sepaqx
```

Install dependencies:

```bash
go mod tidy
```

---

## Code Style

Before committing **always** run:

```bash
./tests/format_check.sh
```

This single check validates formatting for:
- Go (`gofmt`)
- Shell scripts (`shfmt`)
- Docs/text files (trailing whitespace check for `md`, `yml/yaml`, `json`, `toml`, `env`, `txt`)

If `shfmt` is missing:

```bash
go install mvdan.cc/sh/v3/cmd/shfmt@latest
```

---

## Testing

All functional and integration tests are driven by shell scripts.

### Main test entrypoint

The canonical test runner is:

```bash
tests/run.sh
```

This script:
- Starts the service if needed
- Executes API test matrices
- Runs validation checks
- Performs load and stress tests (where applicable)

If `tests/run.sh` passes locally, your change is considered test‑clean.

Do **not** add new test entrypoints unless strictly necessary — extend existing scripts instead.

---

## Commit Guidelines

- Use clear, descriptive commit messages
- One logical change per commit
- Avoid mixing formatting, refactors, and functional changes in one commit

Examples:

```text
fix: validate IBAN length before QR generation
security: harden API key parsing
docs: clarify configuration example
```

---

## Pull Requests

When opening a PR:

1. Ensure `./tests/format_check.sh` passes
2. Ensure `tests/run.sh` passes
3. Explain **what** changed and **why**
4. Reference related issues if applicable

Small, focused PRs are preferred.

---

## Security Issues

Please **do not** report security vulnerabilities via public issues.

Instead, follow the instructions in [`SECURITY.md`](SECURITY.md).

---

## Project Philosophy

SepaQX values:

- Predictability over cleverness
- Explicit configuration over magic
- Clear failure modes
- Minimal dependencies

If a contribution increases complexity, it must also clearly increase robustness or clarity.

---

Thank you for helping make SepaQX better 🚀
