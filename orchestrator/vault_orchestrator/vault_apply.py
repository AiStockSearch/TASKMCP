"""Apply PlanBundle to Vault in one transaction (writer-contract.md)."""

from __future__ import annotations

import hashlib
import json
import uuid
from typing import Any, Optional

import psycopg
from psycopg import Connection
from psycopg.types.json import Json

from vault_orchestrator.models import EpicBlock, PlanBundle, PlanTask

ALLOWED_MB_DOC_TYPES = frozenset(
    {
        "tasks",
        "activeContext",
        "progress",
        "plan",
        "adr",
        "refactor_plan",
        "reflection",
        "archive",
    }
)


def _hash_content(s: str) -> str:
    return hashlib.sha256(s.encode("utf-8")).hexdigest()


def _parse_repo_key(repo_key: str) -> tuple[str, str]:
    parts = [p.strip() for p in repo_key.strip().split("/", 1)]
    if len(parts) != 2 or not parts[0] or not parts[1]:
        raise ValueError("repo_key must be owner/repo")
    return parts[0], parts[1]


def _ensure_project(cur: Any, repo_key: str) -> uuid.UUID:
    owner, name = _parse_repo_key(repo_key)
    cur.execute(
        """
INSERT INTO projects (id, repo_owner, repo_name, repo_key)
VALUES (%s, %s, %s, %s)
ON CONFLICT (repo_key) DO NOTHING
""",
        (str(uuid.uuid4()), owner, name, f"{owner}/{name}"),
    )
    cur.execute("SELECT id FROM projects WHERE repo_key = %s", (f"{owner}/{name}",))
    row = cur.fetchone()
    if not row:
        raise RuntimeError("ensure project failed")
    return uuid.UUID(str(row[0]))


def _ensure_epic(cur: Any, project_id: uuid.UUID, epic: EpicBlock) -> uuid.UUID:
    cur.execute(
        """
SELECT id FROM epics
WHERE project_id = %s AND title = %s
LIMIT 1
""",
        (str(project_id), epic.title),
    )
    row = cur.fetchone()
    if row:
        return uuid.UUID(str(row[0]))
    eid = uuid.uuid4()
    desc = (epic.description or "").strip() or None
    cur.execute(
        """
INSERT INTO epics (id, project_id, title, description, status)
VALUES (%s, %s, %s, %s, %s)
""",
        (str(eid), str(project_id), epic.title, desc, epic.status),
    )
    return eid


def _upsert_requirement(
    cur: Any, project_id: uuid.UUID, title: str, spec: dict, status: str
) -> uuid.UUID:
    rid = uuid.uuid4()
    cur.execute(
        """
INSERT INTO requirements (id, project_id, title, spec_json, status)
VALUES (%s, %s, %s, %s, %s)
ON CONFLICT (project_id, title)
DO UPDATE SET
  spec_json = EXCLUDED.spec_json,
  status = EXCLUDED.status
RETURNING id
""",
        (str(rid), str(project_id), title, Json(spec), status),
    )
    row = cur.fetchone()
    if not row:
        raise RuntimeError("upsert requirement returned no id")
    return uuid.UUID(str(row[0]))


def _insert_task(
    cur: Any,
    project_id: uuid.UUID,
    requirement_id: Optional[uuid.UUID],
    epic_id: Optional[uuid.UUID],
    task: PlanTask,
) -> uuid.UUID:
    desc = (task.description or "").strip() or None

    def _refresh_requirement_task() -> uuid.UUID:
        assert requirement_id is not None
        cur.execute(
            """
SELECT id FROM tasks
WHERE project_id = %s AND requirement_id = %s AND title = %s
LIMIT 1
""",
            (str(project_id), str(requirement_id), task.title),
        )
        row = cur.fetchone()
        if not row:
            raise RuntimeError(f"task dedup miss: {task.title!r}")
        tid_existing = uuid.UUID(str(row[0]))
        cur.execute(
            """
UPDATE tasks
SET description = COALESCE(%s, description),
    priority = %s,
    status = %s,
    epic_id = COALESCE(%s, epic_id)
WHERE id = %s
""",
            (
                desc,
                task.priority,
                task.status,
                str(epic_id) if epic_id else None,
                str(tid_existing),
            ),
        )
        return tid_existing

    def _refresh_epic_only_task() -> uuid.UUID:
        assert epic_id is not None
        cur.execute(
            """
SELECT id FROM tasks
WHERE project_id = %s AND epic_id = %s AND title = %s AND requirement_id IS NULL
LIMIT 1
""",
            (str(project_id), str(epic_id), task.title),
        )
        row = cur.fetchone()
        if not row:
            raise RuntimeError(f"epic task dedup miss: {task.title!r}")
        tid_existing = uuid.UUID(str(row[0]))
        cur.execute(
            """
UPDATE tasks
SET description = COALESCE(%s, description), priority = %s, status = %s
WHERE id = %s
""",
            (desc, task.priority, task.status, str(tid_existing)),
        )
        return tid_existing

    if requirement_id is not None:
        tid = uuid.uuid4()
        cur.execute(
            """
INSERT INTO tasks (
  id, project_id, requirement_id, epic_id, title, description, status, priority
)
VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
ON CONFLICT (project_id, requirement_id, title) WHERE requirement_id IS NOT NULL
DO NOTHING
RETURNING id
""",
            (
                str(tid),
                str(project_id),
                str(requirement_id),
                str(epic_id) if epic_id else None,
                task.title,
                desc,
                task.status,
                task.priority,
            ),
        )
        row = cur.fetchone()
        if row:
            return uuid.UUID(str(row[0]))
        return _refresh_requirement_task()

    if epic_id is not None:
        tid = uuid.uuid4()
        cur.execute(
            """
INSERT INTO tasks (
  id, project_id, requirement_id, epic_id, title, description, status, priority
)
VALUES (%s, %s, NULL, %s, %s, %s, %s, %s)
ON CONFLICT (project_id, epic_id, title) WHERE epic_id IS NOT NULL
DO NOTHING
RETURNING id
""",
            (
                str(tid),
                str(project_id),
                str(epic_id),
                task.title,
                desc,
                task.status,
                task.priority,
            ),
        )
        row = cur.fetchone()
        if row:
            return uuid.UUID(str(row[0]))
        return _refresh_epic_only_task()

    tid = uuid.uuid4()
    cur.execute(
        """
INSERT INTO tasks (
  id, project_id, requirement_id, epic_id, title, description, status, priority
)
VALUES (%s, %s, NULL, NULL, %s, %s, %s, %s)
""",
        (str(tid), str(project_id), task.title, desc, task.status, task.priority),
    )
    return tid


def _insert_task_files(cur: Any, project_id: uuid.UUID, task_id: uuid.UUID, paths: list[str]) -> int:
    n = 0
    for p in paths:
        fp = p.strip()
        if not fp:
            continue
        cur.execute(
            """
INSERT INTO task_files (id, project_id, task_id, file_path)
VALUES (%s, %s, %s, %s)
ON CONFLICT (project_id, task_id, file_path) DO NOTHING
""",
            (str(uuid.uuid4()), str(project_id), str(task_id), fp),
        )
        if cur.rowcount > 0:
            n += 1
    return n


def _upsert_mb_document(cur: Any, project_id: uuid.UUID, doc_key: str, doc_type: str, title: str, content: str) -> None:
    if doc_type not in ALLOWED_MB_DOC_TYPES:
        raise ValueError(f"invalid memory_bank doc_type: {doc_type!r}")
    h = _hash_content(content)
    cur.execute(
        """
SELECT id, current_version
FROM mb_documents
WHERE project_id = %s AND doc_key = %s
FOR UPDATE
""",
        (str(project_id), doc_key),
    )
    row = cur.fetchone()
    if row:
        doc_id = uuid.UUID(str(row[0]))
        current_ver = int(row[1])
    else:
        doc_id = uuid.uuid4()
        cur.execute(
            """
INSERT INTO mb_documents (id, project_id, doc_type, doc_key, title, current_version)
VALUES (%s, %s, %s, %s, NULLIF(%s, ''), 0)
""",
            (str(doc_id), str(project_id), doc_type, doc_key, title),
        )
        current_ver = 0

    if current_ver > 0:
        cur.execute(
            """
SELECT content_hash FROM mb_document_versions
WHERE document_id = %s AND version = %s
""",
            (str(doc_id), current_ver),
        )
        rh = cur.fetchone()
        if rh and str(rh[0]) == h:
            cur.execute(
                """
UPDATE mb_documents
SET doc_type = %s, title = NULLIF(%s, ''), updated_at = now()
WHERE id = %s AND project_id = %s
""",
                (doc_type, title, str(doc_id), str(project_id)),
            )
            return

    new_ver = current_ver + 1
    cur.execute(
        """
INSERT INTO mb_document_versions (id, document_id, version, content, content_hash)
VALUES (%s, %s, %s, %s, %s)
""",
        (str(uuid.uuid4()), str(doc_id), new_ver, content, h),
    )
    cur.execute(
        """
UPDATE mb_documents
SET doc_type = %s, title = NULLIF(%s, ''), current_version = %s, updated_at = now()
WHERE id = %s AND project_id = %s
""",
        (doc_type, title, new_ver, str(doc_id), str(project_id)),
    )


def _upsert_mb_state(cur: Any, project_id: uuid.UUID, state: dict) -> None:
    cur.execute(
        """
INSERT INTO mb_state (project_id, state_json, updated_at)
VALUES (%s, %s::jsonb, now())
ON CONFLICT (project_id) DO UPDATE
SET state_json = EXCLUDED.state_json, updated_at = now()
""",
        (str(project_id), json.dumps(state)),
    )


def apply_plan_bundle(conn: Connection, bundle: PlanBundle) -> dict[str, Any]:
    """Run full writer-contract transaction. Caller manages commit/rollback."""
    summary: dict[str, Any] = {
        "repo_key": bundle.repo_key,
        "tasks_linked": 0,
        "task_files_inserted": 0,
        "memory_bank_docs": 0,
    }
    with conn.cursor() as cur:
        project_id = _ensure_project(cur, bundle.repo_key)
        summary["project_id"] = str(project_id)

        epic_id: Optional[uuid.UUID] = None
        if bundle.epic is not None:
            epic_id = _ensure_epic(cur, project_id, bundle.epic)
            summary["epic_id"] = str(epic_id)

        req = bundle.requirement
        req_id = _upsert_requirement(cur, project_id, req.title, req.spec_json, req.status)
        summary["requirement_id"] = str(req_id)

        files_total = 0
        for task in bundle.tasks:
            req_link = req_id if task.link_to.requirement else None
            epic_link = epic_id if task.link_to.epic else None
            if task.link_to.requirement and req_link is None:
                req_link = req_id
            tid = _insert_task(cur, project_id, req_link, epic_link, task)
            files_total += _insert_task_files(cur, project_id, tid, task.task_files)
        summary["tasks_linked"] = len(bundle.tasks)
        summary["task_files_inserted"] = files_total

        if bundle.memory_bank:
            for dw in bundle.memory_bank.doc_writes:
                _upsert_mb_document(cur, project_id, dw.doc_key, dw.doc_type, dw.title, dw.content)
                summary["memory_bank_docs"] += 1
            if bundle.memory_bank.state_json is not None:
                _upsert_mb_state(cur, project_id, bundle.memory_bank.state_json)

    return summary
