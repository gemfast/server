# Local OTel Collector for gemfast

Routes gemfast-server traces through an OpenTelemetry Collector before they
reach Honeycomb, applying two transforms:

1. **Attribute enrichment** — every span gets `collector.processed=true` and
   `collector.deployment_env=<DEPLOYMENT_ENV>`.
2. **Tail sampling** — full traces are buffered for 5s, then:
   - **Always kept**: traces with any error-status span, any span over 500ms,
     and all gem upload traces (`gem.action=upload`).
   - **Probabilistically kept**: 10% of remaining healthy traces.

## Run

```bash
cd deploy/otel-collector
HONEYCOMB_API_KEY=... DEPLOYMENT_ENV=local docker compose up -d
```

Then point gemfast-server at the local collector instead of Honeycomb:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 \
  ./gemfast-server start -c test/fixtures/gemfast-local.hcl
```

To verify the transform fired: in Honeycomb, run
`COUNT WHERE collector.processed = true` — pre-collector traces won't have the
attribute.

To verify the sampling fired: compare `COUNT` of `GET /up` (cheap, healthy,
high-volume) against `COUNT` of `gem.action=upload` over the same window. The
upload traces should be kept 1:1; the `/up` traces should drop ~90%.

## Stop

```bash
docker compose down
```
