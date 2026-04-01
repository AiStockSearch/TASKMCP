"""CLI: arq worker process."""

from arq import run_worker

from vault_orchestrator.jobs import WorkerSettings


def main() -> None:
    run_worker(WorkerSettings)


if __name__ == "__main__":
    main()
