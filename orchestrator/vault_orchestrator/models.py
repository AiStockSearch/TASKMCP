"""Pydantic models aligned with examples/planbundle_*.json and docs/writer-contract.md."""

from __future__ import annotations

from typing import Any, Optional

from pydantic import BaseModel, Field


class LinkTo(BaseModel):
    requirement: bool = False
    epic: bool = False


PlanTaskFileList = list[str]


class PlanTask(BaseModel):
    title: str
    description: Optional[str] = None
    priority: int = 100
    status: str = "todo"
    link_to: LinkTo = Field(default_factory=LinkTo)
    task_files: PlanTaskFileList = Field(default_factory=list)


class RequirementBlock(BaseModel):
    title: str
    status: str = "todo"
    spec_json: dict[str, Any] = Field(default_factory=dict)


class EpicBlock(BaseModel):
    mode: str = "create_or_skip"
    title: str
    description: Optional[str] = None
    status: str = "open"


class MemoryDocWrite(BaseModel):
    doc_key: str
    doc_type: str
    title: str = ""
    content: str = ""


class MemoryBankBlock(BaseModel):
    doc_writes: list[MemoryDocWrite] = Field(default_factory=list)
    state_json: Optional[dict[str, Any]] = None


class PlanBundle(BaseModel):
    repo_key: str
    requirement: RequirementBlock
    epic: Optional[EpicBlock] = None
    tasks: list[PlanTask] = Field(default_factory=list)
    memory_bank: Optional[MemoryBankBlock] = None
