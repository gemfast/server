# Python clients for gemfast-server

Two scripts that drive the local gemfast-server over real HTTP, primarily
to populate OpenTelemetry traces in Honeycomb.

| Script | Purpose |
|---|---|
| `upload_gem.py` | One-shot cross-language trace propagation demo: login → token → upload a single sample gem. |
| `load_gems.py` | Hour-long mixed-traffic load generator: ~100 direct uploads + ~100 mirror cache fetches + continuous index polling + auth churn across six users. |

## Prerequisites

1. **OTel collector running**, with `HONEYCOMB_API_KEY` exported:
   ```sh
   cd deploy/otel-collector
   HONEYCOMB_API_KEY=hcaik_... DEPLOYMENT_ENV=local docker compose up -d
   ```
2. **gemfast-server running** against the fixture HCL (which seeds the
   six users this script logs in as: `alice`, `carol`, `dave`, `erin`,
   `frank` with role `write` and `bob` with role `read`):
   ```sh
   OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 \
     ./gemfast-server start -c test/fixtures/gemfast-local.hcl
   ```
3. Python deps:
   ```sh
   cd scripts/python-client
   pip install -r requirements.txt
   ```

## `load_gems.py`

```sh
python load_gems.py
```

Default behaviour: 60 minutes, four worker pools, ~200 gem "installs"
total, traces exported to the local collector at `http://localhost:4318`
(the collector adds `x-honeycomb-team` and forwards to Honeycomb).

### Environment variables

| Var | Default | Notes |
|---|---|---|
| `BASE_URL` | `http://localhost:2020` | gemfast-server |
| `RUBYGEMS_API` | `https://rubygems.org` | upstream gem metadata + binary source |
| `DURATION_SECONDS` | `3600` | total wall-clock runtime |
| `UPLOAD_WORKERS` | `2` | direct-upload concurrency |
| `PROXY_WORKERS` | `3` | mirror-cache GET concurrency |
| `INDEX_WORKERS` | `3` | index/metadata GET concurrency |
| `TARGET_UPLOADS` | `100` | shapes per-worker sleep so the run hits this count |
| `TARGET_PROXY_FETCHES` | `100` | same, for proxy fetches |
| `GEM_LIMIT` | `250` | first N entries of `gem_list.py` to draw from |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://localhost:4318` | set to `https://api.honeycomb.io` to send direct (then `HONEYCOMB_API_KEY` is required on the script too) |
| `DEPLOYMENT_ENV` | `local` | resource attribute |
| `LOAD_RUN_ID` | random UUID | tag every span for one run |

### Smoke test

```sh
DURATION_SECONDS=60 TARGET_UPLOADS=3 TARGET_PROXY_FETCHES=5 python load_gems.py
```

Should print a one-line progress summary at the 60s mark and exit 0.

### What to look for in Honeycomb

- `COUNT GROUP BY user.username` — non-zero for all five write users.
- `COUNT WHERE name = "POST /private/api/v1/gems" GROUP BY gem.name` —
  ~100 distinct gems.
- `COUNT WHERE name = "GET /gems/:gem"` — the mirror-cache traffic.
- `COUNT WHERE error.category = "auth"` — from the deliberate
  bad-password attempts in the auth churn worker.
- Trace view on any `client.upload_gem` span — should show child server
  spans from `gemfast-server`, confirming cross-language propagation.

### Local-disk sanity check

After a run, the proxy worker's cache misses leave files on disk:
```sh
ls /tmp/gemfast/gems/rubygems.org/*/ | wc -l
```
should be in the high tens.

## `upload_gem.py`

Single-trace smoke test — see its module docstring.
