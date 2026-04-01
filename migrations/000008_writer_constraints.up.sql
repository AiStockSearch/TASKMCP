CREATE UNIQUE INDEX IF NOT EXISTS uq_requirements_project_title
  ON requirements(project_id, title);

CREATE UNIQUE INDEX IF NOT EXISTS uq_tasks_project_requirement_title
  ON tasks(project_id, requirement_id, title)
  WHERE requirement_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_tasks_project_epic_title
  ON tasks(project_id, epic_id, title)
  WHERE epic_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_task_files_project_task_path
  ON task_files(project_id, task_id, file_path);

CREATE UNIQUE INDEX IF NOT EXISTS uq_github_links_project_entity
  ON github_links(project_id, entity_type, entity_id);
