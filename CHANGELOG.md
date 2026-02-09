# Changelog

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
