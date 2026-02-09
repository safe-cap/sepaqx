# SepaQX Configuration

Version: 0.1.0

Configuration is read from environment variables. `.env` is optional.
For systemd installations, place it at `/etc/sepaqx/.env`.

## Core

- `LISTEN_IP` (default `127.0.0.1`)  
  Address to bind the server instance to. Use `0.0.0.0` to listen on all interfaces.

- `LISTEN_PORT` (default `8089`)

- `KEYS_FILE` (default `./keys.json`)  
  When run as a systemd service, this maps to `/etc/sepaqx/keys.json`.

- `REQUIRE_KEYS` (default `false`)  
  Startup guard: if true, the server refuses to start when `keys.json` cannot be loaded or is empty.

- `REQUIRE_API_KEY` (default `false`)  
  Access control: if true, public access is disabled and every request must include a valid API key from `keys.json`.

Note: `REQUIRE_KEYS` is about startup safety; `REQUIRE_API_KEY` is about request authorization. You can enable both to prevent accidental public mode and to fail fast if keys are missing or invalid.

- `LOGO_MAX_RATIO` (default `0.22`)  
  Maximum logo size ratio, allowed range: `(0, 0.5)`.

## Per-key settings (keys.json)

- `key`  
  API key string.

- `name`  
  Human-friendly label (used in logs).

- `logo_path` (optional)  
  Path to logo PNG. If unreadable, logo is disabled for this key.

- `logo_bg_shape` (default `square`)  
  Background shape behind the logo: `square` or `circle`.

- `palette.fg` / `palette.bg` (optional)  
  Solid foreground/background colors. Hex like `#RRGGBB`.

- `fg_gradient` (optional)  
  Gradient for the foreground (modules). Example: `{ "from": "#7a5cff", "to": "#3aa8ff", "angle": 45 }`.

- `bg_gradient` (optional)  
  Gradient for the background. Example: `{ "from": "#ffffff", "to": "#eef6ff", "angle": 45 }`.

- `corner_radius` (default `0`)  
  Rounded corners on the full PNG image (pixels).

- `module_style` (default `square`)  
  `square`, `rounded`, or `blob`.

- `module_radius` (default `0.25`, allowed range `0..0.5`)  
  Rounded module radius as fraction of module size (used for `rounded` and `blob`).


- `quiet_zone` (default `4`, allowed range `0..20`)  
  Quiet zone (margin) around the QR in module units.

## TLS

- `TLS_ENABLED` (default `false`)  
  When `false`, the server instance runs HTTP (no TLS).

- `TLS_CERT_FILE` (default `./tls/cert.pem`)
- `TLS_KEY_FILE` (default `./tls/key.pem`)

- `TLS_HOSTS` (default `localhost,127.0.0.1`)  
  Comma-separated list of hosts/IPs added to the self-signed certificate.

- `TLS_AUTO_SELF_SIGNED` (default `true`)  
  If `true` and cert/key are missing, a self-signed cert is generated.

- `TLS_CERT_DAYS` (default `365`)  
  Valid range: `1..3650`.

TLS defaults:
- Minimum TLS version: 1.2
- Strong cipher suites for TLS 1.2 (TLS 1.3 uses Go defaults)

## Timeouts and Limits

- `READ_TIMEOUT_SEC` (default `10`)
- `WRITE_TIMEOUT_SEC` (default `15`)
- `IDLE_TIMEOUT_SEC` (default `60`)
- `READ_HEADER_TIMEOUT_SEC` (default `5`)

- `MAX_HEADER_BYTES` (default `1048576`, allowed range `8192..16777216`)
- `MAX_BODY_BYTES` (default `8192`, allowed range `1024..1048576`)

- `RATE_LIMIT_RPS` (default `10`, allowed range `0..1000000`)
- `RATE_LIMIT_BURST` (default `20`, allowed range `1..1000000`)

Timeouts are clamped to `1..600` seconds. Out-of-range values fall back to defaults.

## Validation Limits (API)

- `name`: max 70 characters.
- `purpose`: max 4 characters (uppercased, not strictly validated).
- `remittance_reference`: max 25 characters.
- `remittance_text`: max 140 characters.
- `information`: max 70 characters.
- `amount`: must be > 0 and <= `99999999999` cents.
- `remittance_reference` and `remittance_text` are mutually exclusive.

Rate limiting is per client IP (token bucket with `RATE_LIMIT_RPS` and `RATE_LIMIT_BURST`).

## Legacy Behavior

- `ALLOW_QUERY_API_KEY` (default `false`)  
  Allows `api_key` query parameter as an alternative to `X-API-Key` for GET/HEAD.  
  ⚠️ Strong warning: enabling this leaks the API key via URL surfaces (reverse-proxy access logs, browser history, and referrers). Keep disabled unless you fully control all layers and accept the risk.

## Trusted Proxies

- `TRUSTED_PROXY_CIDRS` (default empty)  
  Comma-separated CIDR list (e.g. `10.0.0.0/8,192.168.0.0/16`).  
  When set, client IP is extracted from `X-Forwarded-For` / `X-Real-IP` only if the immediate peer IP is in this list.

Example (adjust to your environment):

```
TRUSTED_PROXY_CIDRS=127.0.0.1/32,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16
```

## Logging

- `ACCESS_LOG` (default `false`)  
  When enabled, logs each request with method, path, status, latency, and client IP.

## Profiling

- `PPROF_ENABLED` (default `false`)  
  If true, exposes pprof on `PPROF_LISTEN`.

- `PPROF_LISTEN` (default `127.0.0.1:6060`)

Warning: never expose pprof to the public internet. Keep it bound to localhost or a private network.

## Error PNG

- `ERROR_PNG_PATH` (default empty)  
  Optional path to a custom PNG returned on invalid input for QR generation.

## Cache

- `CACHE_PNG_MAX_BYTES` (default `268435456`)
- `CACHE_LOGO_MAX_BYTES` (default `33554432`)
- `CACHE_TTL_SEC` (default `900`)
- `CACHE_CONTROL` (default `private, max-age=60`)

## Project docs

- `CHANGELOG.md`
- `SECURITY.md`
- `LICENSE`
