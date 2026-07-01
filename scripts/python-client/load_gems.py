"""
Gemfast load generator.

Runs four concurrent worker pools against a local gemfast-server for a
configurable duration (default 1 hour):

  - uploader      : JWT login + API token mint per write-user, then
                    downloads a gem from rubygems.org and POSTs it to
                    `/private/api/v1/gems` (Basic auth).
  - proxy_fetcher : anonymous GET `/gems/<name>-<version>.gem` — first hit
                    is a cache miss, server fetches upstream and indexes.
  - index_poller  : anonymous reads against the mirror index endpoints
                    (`/specs.4.8.gz`, `/info/<gem>`, `/versions`, etc.).
  - auth_churn    : background login + refresh-token + bad-password
                    traffic across all users to populate auth telemetry.

Honeycomb plumbing: the script exports its own spans via OTLP/HTTP. By
default it targets the local OTel collector at http://localhost:4318;
the collector is what holds HONEYCOMB_API_KEY and fans traces out to
api.honeycomb.io. If OTEL_EXPORTER_OTLP_ENDPOINT is set directly to
api.honeycomb.io, the script falls back to attaching x-honeycomb-team
itself (matches scripts/python-client/upload_gem.py:32-37).

Run:
    cd scripts/python-client
    pip install -r requirements.txt
    # collector must already be up with HONEYCOMB_API_KEY exported
    python load_gems.py
"""

from __future__ import annotations

import asyncio
import base64
import os
import random
import signal
import sys
import time
import uuid
from dataclasses import dataclass, field
from typing import Optional

import httpx
from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.instrumentation.httpx import HTTPXClientInstrumentor
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor

from gem_list import GEMS


WRITE_USERS = [
    ("alice", "alice-pw"),
    ("carol", "carol-pw"),
    ("dave", "dave-pw"),
    ("erin", "erin-pw"),
    ("frank", "frank-pw"),
]
READ_USERS = [("bob", "bob-pw")]
ALL_USERS = WRITE_USERS + READ_USERS

INDEX_ENDPOINTS = [
    "/specs.4.8.gz",
    "/latest_specs.4.8.gz",
    "/prerelease_specs.4.8.gz",
    "/versions",
]


@dataclass
class Config:
    base_url: str
    rubygems_api: str
    duration_s: int
    upload_workers: int
    proxy_workers: int
    index_workers: int
    target_uploads: int
    target_proxy: int
    gem_limit: int
    deployment_env: str
    run_id: str


@dataclass
class Counters:
    uploaded: int = 0
    proxied: int = 0
    polls: int = 0
    auth_events: int = 0
    errors: int = 0
    lock: asyncio.Lock = field(default_factory=asyncio.Lock)

    async def bump(self, key: str, n: int = 1) -> None:
        async with self.lock:
            setattr(self, key, getattr(self, key) + n)


@dataclass
class GemMeta:
    name: str
    version: str
    platform: str

    @property
    def filename(self) -> str:
        if self.platform and self.platform != "ruby":
            return f"{self.name}-{self.version}-{self.platform}.gem"
        return f"{self.name}-{self.version}.gem"


def load_config() -> Config:
    return Config(
        base_url=os.environ.get("BASE_URL", "http://localhost:2020").rstrip("/"),
        rubygems_api=os.environ.get("RUBYGEMS_API", "https://rubygems.org").rstrip("/"),
        duration_s=int(os.environ.get("DURATION_SECONDS", "3600")),
        upload_workers=int(os.environ.get("UPLOAD_WORKERS", "2")),
        proxy_workers=int(os.environ.get("PROXY_WORKERS", "3")),
        index_workers=int(os.environ.get("INDEX_WORKERS", "3")),
        target_uploads=int(os.environ.get("TARGET_UPLOADS", "100")),
        target_proxy=int(os.environ.get("TARGET_PROXY_FETCHES", "100")),
        gem_limit=int(os.environ.get("GEM_LIMIT", "250")),
        deployment_env=os.environ.get("DEPLOYMENT_ENV", "local"),
        run_id=os.environ.get("LOAD_RUN_ID", str(uuid.uuid4())),
    )


def configure_tracing(cfg: Config) -> trace.Tracer:
    endpoint = os.environ.get("OTEL_EXPORTER_OTLP_ENDPOINT")
    if not endpoint:
        os.environ["OTEL_EXPORTER_OTLP_ENDPOINT"] = "http://localhost:4318"
        endpoint = os.environ["OTEL_EXPORTER_OTLP_ENDPOINT"]
    if "honeycomb.io" in endpoint and not os.environ.get("OTEL_EXPORTER_OTLP_HEADERS"):
        api_key = os.environ.get("HONEYCOMB_API_KEY", "")
        if api_key:
            os.environ["OTEL_EXPORTER_OTLP_HEADERS"] = f"x-honeycomb-team={api_key}"
    os.environ.setdefault("OTEL_SERVICE_NAME", "gemfast-load-generator")

    resource = Resource.create(
        {
            "service.name": os.environ["OTEL_SERVICE_NAME"],
            "deployment.environment": cfg.deployment_env,
            "client.language": "python",
            "load.run_id": cfg.run_id,
            "load.duration_seconds": cfg.duration_s,
        }
    )
    provider = TracerProvider(resource=resource)
    provider.add_span_processor(BatchSpanProcessor(OTLPSpanExporter()))
    trace.set_tracer_provider(provider)
    HTTPXClientInstrumentor().instrument()
    return trace.get_tracer(__name__)


class GemMetaCache:
    """Resolves and caches `<name>` -> GemMeta via rubygems.org public API."""

    def __init__(self, http: httpx.AsyncClient, rubygems_api: str) -> None:
        self._http = http
        self._api = rubygems_api
        self._cache: dict[str, GemMeta] = {}
        self._negative: set[str] = set()
        self._lock = asyncio.Lock()

    async def get(self, name: str) -> Optional[GemMeta]:
        async with self._lock:
            if name in self._cache:
                return self._cache[name]
            if name in self._negative:
                return None
        try:
            resp = await self._http.get(
                f"{self._api}/api/v1/gems/{name}.json", timeout=15.0
            )
        except httpx.HTTPError:
            async with self._lock:
                self._negative.add(name)
            return None
        if resp.status_code != 200:
            async with self._lock:
                self._negative.add(name)
            return None
        data = resp.json()
        meta = GemMeta(
            name=data["name"],
            version=data["version"],
            platform=data.get("platform", "ruby") or "ruby",
        )
        async with self._lock:
            self._cache[name] = meta
        return meta


class GemfastClient:
    """Per-user JWT + API-token holder. Re-mints on a background refresh."""

    def __init__(
        self,
        http: httpx.AsyncClient,
        base_url: str,
        username: str,
        password: str,
    ) -> None:
        self._http = http
        self._base = base_url
        self.username = username
        self._password = password
        self._jwt: Optional[str] = None
        self._api_token: Optional[str] = None
        self._lock = asyncio.Lock()

    async def ensure(self) -> str:
        async with self._lock:
            if self._api_token:
                return self._api_token
            await self._login_locked()
            await self._mint_token_locked()
            assert self._api_token is not None
            return self._api_token

    async def refresh(self) -> None:
        async with self._lock:
            await self._login_locked()
            await self._mint_token_locked()

    async def _login_locked(self) -> None:
        resp = await self._http.post(
            f"{self._base}/admin/api/v1/login",
            json={"username": self.username, "password": self._password},
            timeout=15.0,
        )
        resp.raise_for_status()
        self._jwt = resp.json()["token"]

    async def _mint_token_locked(self) -> None:
        assert self._jwt is not None
        resp = await self._http.post(
            f"{self._base}/admin/api/v1/token",
            headers={"Authorization": f"Bearer {self._jwt}"},
            timeout=15.0,
        )
        resp.raise_for_status()
        self._api_token = resp.json()["token"]

    @property
    def jwt(self) -> Optional[str]:
        return self._jwt

    def basic_header(self) -> dict[str, str]:
        assert self._api_token is not None
        raw = f"{self.username}:{self._api_token}".encode()
        return {"Authorization": f"Basic {base64.b64encode(raw).decode()}"}


class GemPool:
    """Shared name pool: uploaders take exclusively; proxies sample freely."""

    def __init__(self, names: list[str]) -> None:
        self._all = list(names)
        random.shuffle(self._all)
        self._upload_queue = list(self._all)
        self._lock = asyncio.Lock()

    async def take_for_upload(self) -> Optional[str]:
        async with self._lock:
            if not self._upload_queue:
                return None
            return self._upload_queue.pop()

    def sample_for_proxy(self) -> str:
        return random.choice(self._all)


async def upload_worker(
    worker_id: int,
    cfg: Config,
    tracer: trace.Tracer,
    upstream: httpx.AsyncClient,
    gemfast: httpx.AsyncClient,
    clients: list[GemfastClient],
    meta_cache: GemMetaCache,
    pool: GemPool,
    counters: Counters,
    deadline: float,
    mean_sleep_s: float,
) -> None:
    while time.monotonic() < deadline:
        name = await pool.take_for_upload()
        if name is None:
            return
        client = random.choice(clients)
        with tracer.start_as_current_span("client.upload_gem") as span:
            span.set_attribute("worker.kind", "uploader")
            span.set_attribute("worker.id", worker_id)
            span.set_attribute("user.username", client.username)
            span.set_attribute("gem.name", name)
            span.set_attribute("load.run_id", cfg.run_id)
            try:
                meta = await meta_cache.get(name)
                if meta is None:
                    span.set_attribute("error", True)
                    span.set_attribute("exception.slug", "rubygems-meta-missing")
                    await counters.bump("errors")
                    continue
                span.set_attribute("gem.version", meta.version)
                span.set_attribute("gem.platform", meta.platform)

                gem_url = f"{cfg.rubygems_api}/gems/{meta.filename}"
                fetch = await upstream.get(gem_url, timeout=45.0)
                if fetch.status_code != 200:
                    span.set_attribute("error", True)
                    span.set_attribute("rubygems.status_code", fetch.status_code)
                    span.set_attribute("exception.slug", "rubygems-download-failed")
                    await counters.bump("errors")
                    continue
                gem_bytes = fetch.content
                span.set_attribute("gem.size_bytes", len(gem_bytes))

                await client.ensure()
                upload = await gemfast.post(
                    f"{cfg.base_url}/private/api/v1/gems",
                    content=gem_bytes,
                    headers={
                        **client.basic_header(),
                        "Content-Type": "application/octet-stream",
                    },
                    timeout=60.0,
                )
                span.set_attribute("upload.status_code", upload.status_code)
                if upload.status_code >= 400:
                    span.set_attribute("error", True)
                    span.set_attribute(
                        "exception.slug", f"err-upload-{upload.status_code}"
                    )
                    await counters.bump("errors")
                else:
                    await counters.bump("uploaded")
            except Exception as exc:
                span.record_exception(exc)
                span.set_attribute("error", True)
                span.set_attribute("exception.slug", "upload-exception")
                await counters.bump("errors")

        await asyncio.sleep(max(1.0, random.expovariate(1.0 / mean_sleep_s)))


async def proxy_worker(
    worker_id: int,
    cfg: Config,
    tracer: trace.Tracer,
    gemfast: httpx.AsyncClient,
    meta_cache: GemMetaCache,
    pool: GemPool,
    counters: Counters,
    deadline: float,
    mean_sleep_s: float,
) -> None:
    while time.monotonic() < deadline:
        name = pool.sample_for_proxy()
        with tracer.start_as_current_span("client.proxy_fetch") as span:
            span.set_attribute("worker.kind", "proxy_fetcher")
            span.set_attribute("worker.id", worker_id)
            span.set_attribute("gem.name", name)
            span.set_attribute("load.run_id", cfg.run_id)
            try:
                meta = await meta_cache.get(name)
                if meta is None:
                    span.set_attribute("error", True)
                    span.set_attribute("exception.slug", "rubygems-meta-missing")
                    await counters.bump("errors")
                else:
                    span.set_attribute("gem.version", meta.version)
                    span.set_attribute("gem.filename", meta.filename)
                    resp = await gemfast.get(
                        f"{cfg.base_url}/gems/{meta.filename}", timeout=90.0
                    )
                    span.set_attribute("proxy.status_code", resp.status_code)
                    span.set_attribute("proxy.bytes", len(resp.content))
                    if resp.status_code >= 400:
                        span.set_attribute("error", True)
                        span.set_attribute(
                            "exception.slug", f"err-proxy-{resp.status_code}"
                        )
                        await counters.bump("errors")
                    else:
                        await counters.bump("proxied")
            except Exception as exc:
                span.record_exception(exc)
                span.set_attribute("error", True)
                span.set_attribute("exception.slug", "proxy-exception")
                await counters.bump("errors")

        await asyncio.sleep(max(1.0, random.expovariate(1.0 / mean_sleep_s)))


async def index_worker(
    worker_id: int,
    cfg: Config,
    tracer: trace.Tracer,
    gemfast: httpx.AsyncClient,
    pool: GemPool,
    counters: Counters,
    deadline: float,
    mean_sleep_s: float,
) -> None:
    while time.monotonic() < deadline:
        choice = random.random()
        with tracer.start_as_current_span("client.index_poll") as span:
            span.set_attribute("worker.kind", "index_poller")
            span.set_attribute("worker.id", worker_id)
            span.set_attribute("load.run_id", cfg.run_id)
            try:
                if choice < 0.35:
                    gems = ",".join(
                        pool.sample_for_proxy()
                        for _ in range(random.randint(3, 5))
                    )
                    url = f"{cfg.base_url}/api/v1/dependencies?gems={gems}"
                    endpoint = "/api/v1/dependencies"
                elif choice < 0.6:
                    name = pool.sample_for_proxy()
                    url = f"{cfg.base_url}/info/{name}"
                    endpoint = "/info/:gem"
                    span.set_attribute("gem.name", name)
                else:
                    endpoint = random.choice(INDEX_ENDPOINTS)
                    url = f"{cfg.base_url}{endpoint}"
                span.set_attribute("index.endpoint", endpoint)
                resp = await gemfast.get(url, timeout=30.0, follow_redirects=False)
                span.set_attribute("index.status_code", resp.status_code)
                await counters.bump("polls")
            except Exception as exc:
                span.record_exception(exc)
                span.set_attribute("error", True)
                span.set_attribute("exception.slug", "index-exception")
                await counters.bump("errors")

        await asyncio.sleep(max(0.5, random.expovariate(1.0 / mean_sleep_s)))


async def auth_churn_worker(
    cfg: Config,
    tracer: trace.Tracer,
    gemfast: httpx.AsyncClient,
    clients: list[GemfastClient],
    counters: Counters,
    deadline: float,
    mean_sleep_s: float,
) -> None:
    while time.monotonic() < deadline:
        action = random.random()
        with tracer.start_as_current_span("client.auth_churn") as span:
            span.set_attribute("worker.kind", "auth_churn")
            span.set_attribute("load.run_id", cfg.run_id)
            try:
                if action < 0.5:
                    client = random.choice(clients)
                    await client.refresh()
                    span.set_attribute("auth.action", "relogin")
                    span.set_attribute("user.username", client.username)
                elif action < 0.85:
                    client = random.choice(
                        [c for c in clients if c.jwt is not None]
                    )
                    resp = await gemfast.get(
                        f"{cfg.base_url}/admin/api/v1/refresh-token",
                        headers={"Authorization": f"Bearer {client.jwt}"},
                        timeout=10.0,
                    )
                    span.set_attribute("auth.action", "refresh-token")
                    span.set_attribute("user.username", client.username)
                    span.set_attribute("auth.status_code", resp.status_code)
                else:
                    user = random.choice(ALL_USERS)[0]
                    resp = await gemfast.post(
                        f"{cfg.base_url}/admin/api/v1/login",
                        json={"username": user, "password": "wrong-pw"},
                        timeout=10.0,
                    )
                    span.set_attribute("auth.action", "bad-password")
                    span.set_attribute("user.username", user)
                    span.set_attribute("auth.status_code", resp.status_code)
                    if resp.status_code >= 400:
                        span.set_attribute("error.category", "auth")
                await counters.bump("auth_events")
            except Exception as exc:
                span.record_exception(exc)
                span.set_attribute("error", True)
                span.set_attribute("exception.slug", "auth-churn-exception")
                await counters.bump("errors")

        await asyncio.sleep(max(1.0, random.expovariate(1.0 / mean_sleep_s)))


async def reporter(counters: Counters, deadline: float) -> None:
    interval = 60.0
    start = time.monotonic()
    while time.monotonic() < deadline:
        await asyncio.sleep(interval)
        elapsed = int(time.monotonic() - start)
        print(
            f"[load_gems] t={elapsed}s "
            f"uploaded={counters.uploaded} "
            f"proxied={counters.proxied} "
            f"polls={counters.polls} "
            f"auth_events={counters.auth_events} "
            f"errors={counters.errors}",
            flush=True,
        )


async def run() -> int:
    cfg = load_config()
    tracer = configure_tracing(cfg)

    seen: set[str] = set()
    deduped = [g for g in GEMS if not (g in seen or seen.add(g))]
    gem_names = deduped[: cfg.gem_limit]
    pool = GemPool(gem_names)
    counters = Counters()

    deadline = time.monotonic() + cfg.duration_s

    upload_mean = max(5.0, cfg.duration_s * cfg.upload_workers / max(1, cfg.target_uploads))
    proxy_mean = max(5.0, cfg.duration_s * cfg.proxy_workers / max(1, cfg.target_proxy))

    print(
        f"[load_gems] run_id={cfg.run_id} duration={cfg.duration_s}s "
        f"target_uploads={cfg.target_uploads} target_proxy={cfg.target_proxy} "
        f"upload_mean_sleep={upload_mean:.1f}s proxy_mean_sleep={proxy_mean:.1f}s",
        flush=True,
    )

    timeout = httpx.Timeout(60.0, connect=10.0)
    limits = httpx.Limits(max_connections=64, max_keepalive_connections=32)

    async with httpx.AsyncClient(timeout=timeout, limits=limits, http2=False) as gemfast_http, \
               httpx.AsyncClient(timeout=timeout, limits=limits, http2=False) as upstream_http:

        meta_cache = GemMetaCache(upstream_http, cfg.rubygems_api)
        clients = [
            GemfastClient(gemfast_http, cfg.base_url, u, p)
            for (u, p) in WRITE_USERS
        ]

        loop = asyncio.get_running_loop()
        stop = asyncio.Event()

        def _signal() -> None:
            stop.set()

        for sig in (signal.SIGINT, signal.SIGTERM):
            try:
                loop.add_signal_handler(sig, _signal)
            except NotImplementedError:
                pass

        tasks: list[asyncio.Task] = []
        for i in range(cfg.upload_workers):
            tasks.append(asyncio.create_task(
                upload_worker(i, cfg, tracer, upstream_http, gemfast_http,
                              clients, meta_cache, pool, counters, deadline,
                              upload_mean),
                name=f"uploader-{i}",
            ))
        for i in range(cfg.proxy_workers):
            tasks.append(asyncio.create_task(
                proxy_worker(i, cfg, tracer, gemfast_http, meta_cache, pool,
                             counters, deadline, proxy_mean),
                name=f"proxy-{i}",
            ))
        for i in range(cfg.index_workers):
            tasks.append(asyncio.create_task(
                index_worker(i, cfg, tracer, gemfast_http, pool, counters,
                             deadline, 6.0),
                name=f"index-{i}",
            ))
        tasks.append(asyncio.create_task(
            auth_churn_worker(cfg, tracer, gemfast_http, clients, counters,
                              deadline, 45.0),
            name="auth-churn",
        ))
        tasks.append(asyncio.create_task(reporter(counters, deadline),
                                         name="reporter"))

        async def waiter() -> None:
            while time.monotonic() < deadline and not stop.is_set():
                await asyncio.sleep(1.0)

        await waiter()
        for t in tasks:
            t.cancel()
        await asyncio.gather(*tasks, return_exceptions=True)

    print(
        f"[load_gems] DONE run_id={cfg.run_id} "
        f"uploaded={counters.uploaded} "
        f"proxied={counters.proxied} "
        f"polls={counters.polls} "
        f"auth_events={counters.auth_events} "
        f"errors={counters.errors}",
        flush=True,
    )

    provider = trace.get_tracer_provider()
    if hasattr(provider, "shutdown"):
        provider.shutdown()
    return 0


def main() -> int:
    try:
        return asyncio.run(run())
    except KeyboardInterrupt:
        return 130


if __name__ == "__main__":
    sys.exit(main())
