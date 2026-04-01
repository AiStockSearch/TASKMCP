"""FastAPI: enqueue PlanBundle jobs to Redis (arq)."""

from __future__ import annotations

import secrets
from contextlib import asynccontextmanager
from typing import Annotated

from arq import create_pool
from arq.connections import ArqRedis, RedisSettings
from fastapi import Depends, FastAPI, Header, HTTPException, status
from fastapi.responses import JSONResponse
from pydantic import BaseModel, Field

from vault_orchestrator.models import PlanBundle
from vault_orchestrator.settings import Settings, get_settings


class ApplyPlanResponse(BaseModel):
    job_id: str | None = Field(description="arq job id")
    status: str = "queued"


class HealthResponse(BaseModel):
    ok: bool = True


def _verify_bearer(authorization: str | None, secret: str | None) -> None:
    if not secret:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="server missing WEBHOOK_SECRET",
        )
    if not authorization or not authorization.startswith("Bearer "):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="missing or invalid Authorization header",
        )
    token = authorization.removeprefix("Bearer ").strip()
    # compare_digest(str, str) требует ASCII; для произвольных секретов — байты UTF-8.
    if not secrets.compare_digest(
        token.encode("utf-8"),
        secret.encode("utf-8"),
    ):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="invalid token",
        )


@asynccontextmanager
async def lifespan(app: FastAPI):
    settings = get_settings()
    if not (settings.webhook_secret and settings.webhook_secret.strip()):
        raise RuntimeError("WEBHOOK_SECRET must be set for the API process")
    pool = await create_pool(RedisSettings.from_dsn(settings.redis_url))
    app.state.redis = pool
    yield
    await pool.close()


app = FastAPI(
    title="Vault orchestrator",
    description="Webhook → Redis (arq) → apply PlanBundle to Postgres Vault",
    lifespan=lifespan,
)


def settings_dep() -> Settings:
    return get_settings()


@app.get("/healthz", response_model=HealthResponse)
async def healthz() -> HealthResponse:
    return HealthResponse()


@app.post(
    "/hooks/apply-plan-bundle",
    status_code=status.HTTP_202_ACCEPTED,
    response_model=ApplyPlanResponse,
)
async def apply_plan_bundle_hook(
    body: PlanBundle,
    authorization: Annotated[str | None, Header()] = None,
    settings: Settings = Depends(settings_dep),
) -> ApplyPlanResponse | JSONResponse:
    _verify_bearer(authorization, settings.webhook_secret)

    redis: ArqRedis = app.state.redis
    payload = body.model_dump(mode="json")
    job = await redis.enqueue_job("process_plan_bundle", payload)
    if job is None:
        return JSONResponse(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            content={"detail": "queue unavailable"},
        )
    return ApplyPlanResponse(job_id=str(job.job_id), status="queued")


def main() -> None:
    s = get_settings()
    import uvicorn

    uvicorn.run(
        "vault_orchestrator.main:app",
        host=s.api_host,
        port=s.api_port,
        factory=False,
    )


if __name__ == "__main__":
    main()
