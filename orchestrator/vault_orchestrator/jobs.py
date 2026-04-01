"""arq background jobs (Redis broker)."""

from __future__ import annotations

import asyncio
import logging
import os
from typing import Any

import psycopg
from arq.connections import RedisSettings

from vault_orchestrator.models import PlanBundle
from vault_orchestrator.settings import get_settings
from vault_orchestrator.vault_apply import apply_plan_bundle

logger = logging.getLogger(__name__)


def _apply_bundle_sync(payload: dict) -> dict[str, Any]:
    settings = get_settings()
    bundle = PlanBundle.model_validate(payload)
    with psycopg.connect(settings.database_url, autocommit=False) as conn:
        summary = apply_plan_bundle(conn, bundle)
    return summary


async def process_plan_bundle(ctx: dict[str, Any], payload: dict) -> dict[str, Any]:
    """Apply PlanBundle JSON to Vault (runs DB work in a thread pool)."""
    try:
        summary = await asyncio.to_thread(_apply_bundle_sync, payload)
    except Exception:
        logger.exception("plan bundle apply failed")
        raise
    logger.info(
        "plan bundle applied repo_key=%s requirement_id=%s tasks=%s",
        summary.get("repo_key"),
        summary.get("requirement_id"),
        summary.get("tasks_linked"),
    )
    return summary


class WorkerSettings:
    redis_settings = RedisSettings.from_dsn(
        os.environ.get("REDIS_URL", "redis://127.0.0.1:6379/0")
    )
    functions = [process_plan_bundle]
