# syntax=docker/dockerfile:1
FROM python:3.12-slim

WORKDIR /app

RUN pip install --no-cache-dir --upgrade pip

COPY pyproject.toml README.md ./
COPY vault_orchestrator ./vault_orchestrator

RUN pip install --no-cache-dir .

ENV PYTHONUNBUFFERED=1

EXPOSE 8080

CMD ["uvicorn", "vault_orchestrator.main:app", "--host", "0.0.0.0", "--port", "8080"]
