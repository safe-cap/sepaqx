# SepaQX
<p align="center">
  <a href="https://safe-cap.com/sepaqx/" target="_blank">
    <img src="img/sepaqx-logo-flat.png" alt="SepaQX">
  </a>
  <br><br>
  <img src="https://img.shields.io/github/v/release/safe-cap/sepaqx?display_name=tag&sort=semver&style=for-the-badge">
  <a href="https://www.paypal.com/donate/?hosted_button_id=JNFS79EFEM7C6" target="_blank">
    <img src="https://img.shields.io/badge/Donate-PayPal-blue.svg?style=for-the-badge">
  </a>
</p>

---

## 💡 Project Overview

**SepaQX** is a lightweight HTTP/HTTPS server instance for generating **SEPA EPC QR codes** (SCT – SEPA Credit Transfer) as PNG images.  
It is designed to be embedded into billing systems, invoices, web applications, and internal tools where fast, compliant SEPA payment QR generation is required.

The server instance supports both **public mode** (no authentication) and **API-key based access** with per-key customization.

---

## 🧾 Why SepaQX exists

SEPA QR generation is deceptively simple — until it is embedded into real billing workflows.

SepaQX was created to solve practical problems that arise in production environments:

- inconsistent EPC implementations across libraries  
- missing validation and unclear error handling  
- lack of branding and per-client customization  
- difficulty embedding QR generation into invoice pipelines  
- security and rate-limiting concerns when exposed publicly  

Instead of providing yet another library, SepaQX runs as a **small, hardened server instance** that can be integrated via HTTP into any system, regardless of language or framework.

---

## 📚 Table of Contents

- [💡 Project Overview](#-project-overview)
- [🏦 What is SepaQX?](#-what-is-sepaqx)
- [⚙️ Features](#-features)
- [🚀 API Usage](#-api-usage)
- [🔑 API Keys & Customization](#-api-keys--customization)
- [📁 Server Instance Runtime & Files](#-server-instance-runtime--files)
- [🛠 Configuration](#-configuration)
- [🔒 Security Notes](#-security-notes)
- [✅ Requirements](#-requirements)
- [📜 License](#-license)
- [🤝 Author](#-author)

---

## 🏦 What is SepaQX?

SepaQX is a **self-hosted SEPA QR code generator** that produces EPC-compliant payment payloads and renders them as QR codes.

When a user scans the QR code (e.g. with **Revolut**, **N26**, **ING**, or other European banking apps), the payment form is pre-filled with:

- Receiver name  
- IBAN / BIC  
- Amount  
- Purpose code  
- Remittance reference or text  

This eliminates manual input errors and significantly speeds up SEPA payments.

---

## ⚙️ Features

- **EPC069-12 compliant SEPA QR codes**
- One endpoint → **PNG output**
- Designed for **invoice & billing systems**
- Public mode or **API-key based access**
- Per-key branding and layout customization:
  - logo overlay
  - colors and gradients
  - module style and shape
- Built-in validation (IBAN, EPC fields)
- Hardened HTTP server:
  - request limits
  - rate limiting
  - caching
- Runs as a **standalone systemd service**
- No external dependencies
- No payment execution

---

## Feature comparison

| Feature | SepaQX | EPC QR libraries | ERP / CMS plugins | Online generators |
| --- | --- | --- | --- | --- |
| Open Source | ✅ | ✅ | ⚠️ depends | ❌ |
| Standalone service | ✅ | ❌ | ❌ | ❌ |
| HTTP API | ✅ | ❌ | ❌ | ⚠️ limited |
| PNG output | ✅ | ⚠️ depends | ✅ | ✅ |
| Designed for invoices | ✅ | ❌ | ⚠️ platform-bound | ⚠️ manual |
| Public + API-key mode | ✅ | ❌ | ❌ | ❌ |
| Per-client branding | ✅ | ❌ | ⚠️ limited | ⚠️ UI only |
| Validation (IBAN/EPC) | ✅ | ⚠️ partial | ⚠️ partial | ⚠️ unclear |
| Error handling (PNG + JSON) | ✅ | ❌ | ❌ | ❌ |
| Rate limiting / limits | ✅ | ❌ | ❌ | ❌ |
| Reverse-proxy aware | ✅ | ❌ | ❌ | ❌ |
| Self-hosted | ✅ | ✅ | ⚠️ | ❌ |
| Language-agnostic | ✅ | ❌ | ❌ | ✅ |
| Production-ready config | ✅ | ❌ | ❌ | ❌ |

---

## ✅ Requirements

### Build from source
- Go 1.22+
- Linux (systemd)

### Install from APT
- Linux (systemd)

---

## 🔐 Supply-chain trust

- Release artifacts include `SHA256SUMS` and `SHA256SUMS.sig` (cosign keyless signatures).
- SBOM is published as CycloneDX JSON in release artifacts.
- CI runs CodeQL, govulncheck, gosec, and staticcheck.

Verify checksums (keyless cosign):

```bash
cosign verify-blob \
  --certificate-identity "https://github.com/safe-cap/sepaqx/.github/workflows/release.yml@refs/heads/main" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  --signature SHA256SUMS.sig \
  SHA256SUMS
```

Then verify a binary against the checksum:

```bash
sha256sum -c SHA256SUMS
```

---

## 📦 Install via APT (our server)

If you do not want to compile it yourself, use our official APT repository:

```
https://install.safe-cap.com/linux/apt
```

Install it like this:

```
curl -fsSL https://install.safe-cap.com/linux/apt/pubkey.gpg | sudo gpg --dearmor -o /usr/share/keyrings/safe-cap.gpg
echo "deb [signed-by=/usr/share/keyrings/safe-cap.gpg] https://install.safe-cap.com/linux/apt stable main" | sudo tee /etc/apt/sources.list.d/safe-cap.list
sudo apt update
sudo apt install sepaqx
```

Repository metadata:

```
Hit: https://install.safe-cap.com/linux/apt stable InRelease
```

**Notes:**
- The APT repository is published via an external pipeline on our infrastructure. For security and operational privacy, this repository does not include the private publication scripts or signing keys.
- Publicly available steps are limited to reproducible build configuration; this repository does not include `.deb` packaging.
- The external pipeline follows the standard flow: build binaries → package `.deb` → sign → publish to the APT repository.

---

## 🧱 Build from Git and run as a server instance

Requires Go 1.22+ and systemd.

```
git clone <your-git-url> sepaqx
cd sepaqx
go build -o sepaqx .
sudo install -m 0755 sepaqx /usr/bin/sepaqx
sudo install -d /etc/sepaqx /etc/sepaqx/tls
sudo cp examples/.env.example /etc/sepaqx/.env
```

Create a systemd unit:

Option A (recommended) — install from the repository:

```
# 1) Create user/group
sudo useradd --system --home /etc/sepaqx --shell /usr/sbin/nologin sepaqx || true

# 2) Create dirs
sudo mkdir -p /etc/sepaqx /var/lib/sepaqx /var/log/sepaqx
sudo chown -R sepaqx:sepaqx /etc/sepaqx /var/lib/sepaqx /var/log/sepaqx

# 3) Install service file from repo checkout
sudo install -m 0644 packaging/systemd/sepaqx.service /etc/systemd/system/sepaqx.service

# 4) Reload + enable + start
sudo systemctl daemon-reload
sudo systemctl enable --now sepaqx

# 5) Check status/logs
systemctl status sepaqx --no-pager
journalctl -u sepaqx -f
```

Notes:
- `.env` path: `/etc/sepaqx/.env`
- Binary path: `/usr/bin/sepaqx`
- TLS key permissions (if applicable):

```
sudo chmod 600 /etc/sepaqx/*.key 2>/dev/null || true
sudo chown sepaqx:sepaqx /etc/sepaqx/*.key 2>/dev/null || true
```

Option B (fallback / quick install) — heredoc:

```
sudo tee /etc/systemd/system/sepaqx.service >/dev/null <<'EOF'
[Unit]
Description=SepaQX server instance
After=network.target

[Service]
ExecStart=/usr/bin/sepaqx
WorkingDirectory=/etc/sepaqx
EnvironmentFile=/etc/sepaqx/.env
Restart=on-failure
RestartSec=2

[Install]
WantedBy=multi-user.target
EOF
```

Enable and start:

```
sudo systemctl daemon-reload
sudo systemctl enable --now sepaqx
```

---

## 🚀 API Usage

### POST (recommended)

```
POST /sepa-qr
Content-Type: application/json
X-API-Key: <optional>
```

```json
{
  "name": "Receiver Name",
  "iban": "DE12345678901234567890",
  "bic": "BANKDEFFXXX",
  "amount": "12.34",
  "purpose": "SALA",
  "remittance_reference": "RF18539007547034",
  "remittance_text": "Invoice 123",
  "information": "Sample EPC QR code"
}
```

If `purpose` is omitted, the default is `GDDS`.

### Limits and Validation

All validation limits and rate-limit semantics are documented in `CONFIG.md` (see “Validation Limits (API)”).

### GET

```
/sepa-qr?name=...&iban=...&bic=...&amount=...&purpose=...&remittance_reference=...&remittance_text=...&information=...&api_key=...
```

### HEAD (public mode only)

If public access is allowed, a bare `HEAD /sepa-qr` (no query string) returns PNG headers only without validating input. This is useful for simple health/probe checks.

### Validate (JSON only)

```
POST /sepa-qr/validate
Content-Type: application/json
```

Returns:

```
{"ok":true,"request_id":"..."}
```

or

```
{"ok":false,"error_code":"invalid_input","details":"...","field":"iban","request_id":"..."}
```

### Error PNG (invalid input)

If input is invalid for QR generation, the server returns a single static PNG error image.
You can override it with `ERROR_PNG_PATH`.

To receive JSON error codes instead of PNG:

```
/sepa-qr?...&format=json
```

Content negotiation:
- `Accept: application/json` → JSON error response
- `Accept: image/png` → PNG error response

JSON error schema:

```
{"ok":false,"error_code":"...","details":"...","field":"...","request_id":"..."}
```

### Health/Ready/Version

```
GET /healthz
GET /readyz
GET /version
```

`/version` returns JSON with build metadata and key config flags.

---

## 🧪 Example request (fake data)

```
curl -X POST http://127.0.0.1:8089/sepa-qr \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Example GmbH",
    "iban": "DE12500105170648489890",
    "bic": "INGDDEFFXXX",
    "amount": "49.90",
    "purpose": "GDDS",
    "remittance_text": "Order 2026-0001",
    "information": "Sample EPC QR code"
  }' --output sepa-qr.png
```

---

## 🧪 Tests

Unified entry point:

```
./tests/run.sh
```

Common options:
- `./tests/run.sh matrix`
- `./tests/run.sh load`
- `./tests/run.sh load-wrk`
- `./tests/run.sh load-hey`

---

## 📘 EPC QR standard (fields)

Supported standard:
- EPC QR Code (SEPA Credit Transfer / SCT):  
  https://en.wikipedia.org/wiki/EPC_QR_code  

Fixed values used by SepaQX:
- Service Tag: `BCD`
- Version: `001`
- Character set: `1`
- Identification: `SCT`

Supported fields:
- `bic`: BIC
- `name`: Creditor name
- `iban`: IBAN
- `amount`: `EUR<amount>` (e.g. `EUR1`)
- `purpose`: 4 characters max (Reason / purpose code)
- `remittance_reference`: Reference of invoice (structured) or empty line
- `remittance_text`: Remittance text (unstructured) or empty line
- `information`: Additional information (e.g. "Sample EPC QR code") or empty line

Note: `remittance_reference` and `remittance_text` are mutually exclusive.

---

## ✅ IBAN validation

SepaQX validates IBANs using the ISO 13616 algorithm (MOD97-10):  
move the first 4 characters to the end, convert letters to numbers (A=10..Z=35), then check that the remainder mod 97 equals 1.  
Only characters A–Z and 0–9 are allowed; length must be 15–34 characters.  
https://en.wikipedia.org/wiki/International_Bank_Account_Number

---

## 🔑 API Keys & Customization

```json
{
  "keys": [
    {
      "key": "my-secret-api-key-1",
      "name": "client-a",
      "logo_path": "/opt/sepaqx/assets/logo-a.png",
      "logo_bg_shape": "circle",
      "palette": {
        "fg": "#000000",
        "bg": "#ffffff"
      },
      "fg_gradient": { "from": "#7a5cff", "to": "#3aa8ff", "angle": 45 },
      "bg_gradient": { "from": "#ffffff", "to": "#eef6ff", "angle": 45 },
      "corner_radius": 24,
      "module_style": "blob",
      "module_radius": 0.5,
      "quiet_zone": 2
    }
  ]
}
```

All optional per-key fields are demonstrated in `examples/keys.json.example`.
Full per-key configuration reference is in `CONFIG.md`.

Use the key to apply client options:

```
curl -X POST http://127.0.0.1:8089/sepa-qr \
  -H "Content-Type: application/json" \
  -H "X-API-Key: my-secret-api-key-1" \
  -d '{
    "name": "Example GmbH",
    "iban": "DE12500105170648489890",
    "bic": "INGDDEFFXXX",
    "amount": "49.90"
  }' --output sepa-qr.png
```

Disable public access (keys-only mode):

```
REQUIRE_API_KEY=true
```

Tip: pair with `REQUIRE_KEYS=true` to fail fast if `keys.json` is missing or empty.

`REQUIRE_KEYS` is a startup guard (refuse to start if keys are missing/empty), while `REQUIRE_API_KEY` controls request authorization (no public access).

---

## 📁 Server Instance Runtime & Files

- Binary: `/usr/bin/sepaqx`
- Config: `/etc/sepaqx`
- TLS: `/etc/sepaqx/tls`
- systemd: `/lib/systemd/system/sepaqx.service`

Logs:

```
journalctl -u sepaqx
```

---

## 🛠 Configuration

Optional `.env` file:

```
/etc/sepaqx/.env
```

Full reference: `CONFIG.md`

Key options:
- `ERROR_PNG_PATH` (custom error PNG for invalid input)
- `PPROF_ENABLED` / `PPROF_LISTEN` (optional profiling)
- `ALLOW_QUERY_API_KEY` (allows `api_key` in GET/HEAD)  
  ⚠️ Strong warning: enabling this leaks the API key via URL surfaces (reverse-proxy access logs, browser history, and referrers). Keep disabled unless you fully control all layers and accept the risk.
- `TRUSTED_PROXY_CIDRS` (trusted reverse proxy CIDRs for real client IP)
- `REQUIRE_KEYS` (fail startup if keys are missing/empty)
- `REQUIRE_API_KEY` (disable public access; require a valid API key)

Example:

```
TRUSTED_PROXY_CIDRS=127.0.0.1/32,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16
```

**PPROF warning:** never expose the pprof listener to the public internet. Default bind is `127.0.0.1:6060`.

---

## 📄 Project files

- `CHANGELOG.md`
- `SECURITY.md`
- `LICENSE`
- `examples/.env.example`
- `examples/keys.json.example`

## 🔒 Security Notes

- HTTPS recommended for public deployments

## 🔧 API Stability

The `/sepa-qr` and `/sepa-qr/validate` contracts are intended to be stable and backwards compatible. New endpoints will be additive.
- API keys optional but recommended
- No payment execution
- No sensitive data stored
- Consistent machine-readable responses for validation endpoints

---

## 💙 Support the project

SepaQX is **free and open-source**.

If it helped you integrate SEPA QR payments faster, avoid errors in invoices, or simplify your billing workflow, supporting the project is a great way to give back.

If SepaQX is used as part of a **commercial product or billing workflow**, your support helps ensure long-term stability and continued development.

Your donation helps with:
- long-term maintenance
- standards compliance updates
- security improvements
- documentation and examples
- keeping the project independent

<p align="center">
  <a href="https://www.paypal.com/donate/?hosted_button_id=JNFS79EFEM7C6" target="_blank">
    <img src="https://img.shields.io/badge/Donate-PayPal-blue.svg?style=for-the-badge">
  </a>
</p>

Commercial support and custom integrations are available on request.

---

## 📜 License

Apache-2.0

---

If you use SepaQX in production, consider adding it to your documentation or internal tooling notes.
Pull requests improving documentation or examples are always welcome.

---

## 🤝 Author
<p align="center">
  <a href="https://safe-cap.com" target="_blank">
    <img src="img/safe-cap-short.png" width="100" height="100" alt="SAFE-CAP">
  </a>
  <br><br>
  <strong>Maintained by SAFE-CAP / Alexander Schiemann</strong><br>
  https://safe-cap.com
</p>
