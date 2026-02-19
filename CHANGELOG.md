# Changelog

## [0.1.2] - 2026-02-19
- Tests: added dedicated backward-compatibility suite (tests/compatibility.sh) for stable API/CLI contract checks.
- Tests: extended tests/run.sh with compatibility mode and updated bitmask mapping (compatibility=8 in all mode).
- CI: added ./tests/run.sh compatibility as a required check in ci.yml.
- CI: nightly full gate now exports artifacts (format_check.log, go_test.log, run_all.log) and publishes a summary in GitHub Actions.
- Tests: expanded amount/OCR fuzz property coverage with profile agreement and auto-lenient agreement invariants.
- Docs: improved README onboarding with a balanced “Get Value in 2 Minutes” entry path and practical use-case framing.
- Docs: added two ready-to-copy integration snippets (Invoice/PDF query mode and internal API-key + validate flow).
- Docs/Wiki: synchronized compatibility policy wording and exit-code mapping (compatibility=8) across README and wiki AI contract.
- Wiki: added short “Start in 2 Minutes” entry section on Home.

## [0.1.1] - 2026-02-19
- Validation: improved `amount` parsing for EUR inputs (`30.12`, `30,12`, `EUR 30.12`, `30,12 €`) and explicit rejection of non-EUR markers (e.g. `$`, `USD`).
- Validation: added optional `AMOUNT_LENIENT_OCR` mode for OCR-like noisy EUR amount formats while keeping non-EUR rejection.
- API/CLI: added optional `scheme` field/flag (default `epc_sct`) to prepare future multi-scheme support; currently only `epc_sct` is accepted.
- API/CLI: added optional `amount_format` (`--amount-format` in CLI) for explicit amount parsing profiles (`eur_dot`, `eur_comma`, grouped variants, `auto_eur_lenient`) to improve noisy/OCR integration reliability.
- CLI: batch mode now returns a non-zero exit code when at least one item fails, while still printing full per-item results.
- CLI: batch JSON output now includes top-level summary fields (`ok`, `total`, `succeeded`, `failed`) for automation/AI orchestration.
- Tests: added CLI tests for JSON batch output, partial-failure batch behavior, and `--format png` batch restriction for `--out -`.
- Tests: expanded Unicode validation coverage (Cyrillic, Latin with diacritics, Greek, Japanese) and added rune-safe truncation checks.
- Tests: added CLI end-to-end integration script (`tests/cli_e2e.sh`) and wired it into `tests/run.sh`.
- Tests: improved `tests/run.sh` output ergonomics in `all` mode (suite totals hidden in the middle, section completion lines + unified final summary).
- Tests: added machine-readable test summary output via `tests/run.sh --json`.
- Tests: added explicit suite-aware exit code mapping in `tests/run.sh` (`1` matrix, `2` cli-e2e, `4` load; bitmask in `all` mode).
- Quality: added `tests/format_check.sh` and CI enforcement for formatting checks across Go, shell, and docs/text files.
- CI: added quality gates for `./tests/run.sh matrix` and `./tests/run.sh cli-e2e` in pull request checks.
- CI: added nightly/full gate workflow (`./tests/run.sh all`) on schedule and pushes to `main`.
- Ops: added `RELEASE_CHECKLIST.md` for repeatable release hygiene.
- Tests: expanded amount/OCR fuzz/property coverage for format profiles, noisy inputs, and conflicting currency markers.
- Ops/API: `/readyz` now reflects strict readiness (`200` when ready, `503` with reason when not ready), including JSON response when `Accept: application/json`.
- Docs: clarified CLI-only install from source (any OS) vs Debian/Ubuntu server install.
- Docs: refined README/wiki structure and synced README/CONFIG with current behavior (`ALLOW_QUERY_API_KEY` conditions, `/sepa-qr/validate` auth behavior, current CLI JSON fields, amount format rules).
- Examples: added `ALLOW_QUERY_API_KEY=false` to `examples/.env.example` with a safety note.

## [0.1.0] - 2026-02-08
- Added `POST /sepa-qr/validate` for JSON-only validation.
- Invalid QR inputs now return a static error PNG (customizable via `ERROR_PNG_PATH`).
- Added optional pprof support via `PPROF_ENABLED` and `PPROF_LISTEN`.
- Unified test runner (`tests/run.sh`) with load options and metrics logging.
- Log/metrics rotation in load-test scripts (`tests/`, keeps last 7 files).
- Increased allowed ranges for rate limit settings.
- GET/HEAD are always enabled and `api_key` is controlled by `ALLOW_QUERY_API_KEY`.
- Added per-key QR styling options: rounded corners and rounded modules.
- Added per-key gradient fills for foreground/background and blob-style modules.
- Added per-key `quiet_zone` to control QR margin size.
- Added per-key `logo_bg_shape` to render logo background as circle or square.
- Added trusted proxy CIDR support for correct client IP behind reverse proxies.
- Added TLS hardening defaults (minimum TLS version and cipher suites).
- Standardized `/sepa-qr` errors to always return error PNG.
- Removed per-response `Last-Modified` header for better caching semantics.
- Added `format=json` option on `/sepa-qr` to return JSON error codes instead of PNG.
- Adjusted custom QR rendering to use full canvas size (quiet zone no longer inflates margins).
- Added `/healthz`, `/readyz`, and `/version` endpoints for service checks and build metadata.
- Added content negotiation for `/sepa-qr` error responses (Accept header).
- Added unified JSON error schema with `request_id` and `error_code` for validation and JSON errors.
- Documented API limits and validation constraints in README/CONFIG.
- Added `REQUIRE_KEYS` to prevent accidental public mode on startup.
- Added `REQUIRE_API_KEY` to disable public access and require valid API keys.
