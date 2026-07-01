"""
Cross-language trace propagation demo.

Starts a root span in `gemfast-python-client`, mints a JWT against the local
gemfast server, fetches a token, and uploads a sample gem — all under one
trace. The `requests` auto-instrumentation injects the W3C `traceparent`
header on every outbound call, so the gemfast Go server picks up the parent
trace and emits its own server span as a child.

Run from the repo root:

    cd scripts/python-client
    pip install -r requirements.txt
    BASE_URL=http://localhost:2020 \
      HONEYCOMB_API_KEY=... \
      python upload_gem.py
"""

import os
import sys

import requests
from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.instrumentation.requests import RequestsInstrumentor
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor


def configure_tracing() -> trace.Tracer:
    if not os.environ.get("OTEL_EXPORTER_OTLP_ENDPOINT"):
        os.environ["OTEL_EXPORTER_OTLP_ENDPOINT"] = "https://api.honeycomb.io"
    if not os.environ.get("OTEL_EXPORTER_OTLP_HEADERS"):
        api_key = os.environ.get("HONEYCOMB_API_KEY", "")
        if api_key:
            os.environ["OTEL_EXPORTER_OTLP_HEADERS"] = f"x-honeycomb-team={api_key}"
    if not os.environ.get("OTEL_SERVICE_NAME"):
        os.environ["OTEL_SERVICE_NAME"] = "gemfast-python-client"

    resource = Resource.create(
        {
            "service.name": os.environ["OTEL_SERVICE_NAME"],
            "deployment.environment": os.environ.get("DEPLOYMENT_ENV", "local"),
            "client.language": "python",
        }
    )
    provider = TracerProvider(resource=resource)
    provider.add_span_processor(BatchSpanProcessor(OTLPSpanExporter()))
    trace.set_tracer_provider(provider)
    RequestsInstrumentor().instrument()  # auto-inject traceparent on every requests call
    return trace.get_tracer(__name__)


def main() -> int:
    base_url = os.environ.get("BASE_URL", "http://localhost:2020").rstrip("/")
    gem_path = os.environ.get(
        "SAMPLE_GEM", "../../test/fixtures/spec/nokogiri-1.15.3-arm64-darwin.gem"
    )
    username = os.environ.get("GEMFAST_USER", "alice")
    password = os.environ.get("GEMFAST_PASSWORD", "alice-pw")

    if not os.path.exists(gem_path):
        print(f"sample gem not found at {gem_path}", file=sys.stderr)
        return 1

    tracer = configure_tracing()

    with tracer.start_as_current_span("client.upload_gem") as span:
        span.set_attribute("client.language", "python")
        span.set_attribute("gem.path", gem_path)
        span.set_attribute("gemfast.base_url", base_url)
        span.set_attribute("gemfast.username", username)

        # 1. Login → JWT
        login_resp = requests.post(
            f"{base_url}/admin/api/v1/login",
            json={"username": username, "password": password},
            timeout=10,
        )
        login_resp.raise_for_status()
        jwt = login_resp.json()["token"]
        span.add_event("login.complete")

        # 2. Mint API token
        token_resp = requests.post(
            f"{base_url}/admin/api/v1/token",
            headers={"Authorization": f"Bearer {jwt}"},
            timeout=10,
        )
        token_resp.raise_for_status()
        api_token = token_resp.json()["token"]
        span.add_event("api_token.minted")

        # 3. Upload gem
        with open(gem_path, "rb") as gem_file:
            upload_resp = requests.post(
                f"{base_url}/private/api/v1/gems",
                data=gem_file.read(),
                auth=(username, api_token),
                headers={"Content-Type": "application/octet-stream"},
                timeout=30,
            )
        span.set_attribute("upload.status_code", upload_resp.status_code)
        span.set_attribute("upload.response_size_bytes", len(upload_resp.content))
        span.add_event("upload.complete")

        if upload_resp.status_code >= 400:
            span.set_attribute("error", True)
            span.set_attribute(
                "exception.slug", f"err-py-upload-{upload_resp.status_code}"
            )
            print(
                f"upload failed: {upload_resp.status_code} {upload_resp.text}",
                file=sys.stderr,
            )
            return 1

        print(f"OK uploaded {gem_path} ({upload_resp.status_code})")
        return 0


if __name__ == "__main__":
    sys.exit(main())
