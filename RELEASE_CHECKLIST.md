# Release Checklist

Use this checklist before creating a release tag.

## Quality gates

- `./tests/format_check.sh` passes
- `go test ./...` passes
- `./tests/run.sh matrix` passes
- `./tests/run.sh cli-e2e` passes
- Optional: `./tests/run.sh load` for performance smoke

## Documentation sync

- `README.md` reflects current product entry points
- Wiki pages updated for changed behavior/contracts
- `CONFIG.md` updated for new environment variables
- `examples/.env.example` updated for new runtime flags

## Change tracking

- `CHANGELOG.md` updated in current release section
- Version references updated where required

## Packaging and metadata

- Build metadata (`version`, `commit`) verified in release pipeline
- Release artifacts/checksums/signatures generated and verified

## Final sanity

- `/health`, `/readyz`, `/version` verified on a fresh instance
- One API QR generation and one CLI generation tested manually
