DO $$
DECLARE
  ws_id bigint;
  usr_id bigint;
  proj_id bigint;
  spr_id bigint;
  old_spr_id bigint;
  rel_id bigint;
BEGIN
  SELECT w.id INTO ws_id FROM workspaces w WHERE w.name = '1984 Demo' LIMIT 1;
  IF ws_id IS NULL THEN
    INSERT INTO workspaces (name) VALUES ('1984 Demo') RETURNING id INTO ws_id;
  END IF;

  SELECT u.id INTO usr_id FROM users u WHERE u.workspace_id = ws_id AND u.email = 'demo@1984.local';
  IF usr_id IS NULL THEN
    INSERT INTO users (workspace_id, name, email, password_hash, role)
    VALUES (ws_id, 'Demo 1984', 'demo@1984.local',
      'pbkdf2-sha256$600000$MTk4NC1kZW1vLXNhbHQhIQ$kg3TGh4GkVp7HMY+qo5vqIGaywrrtL6zLbEdflqsJqU', 'admin')
    RETURNING id INTO usr_id;
  END IF;

  SELECT p.id INTO proj_id FROM projects p WHERE p.workspace_id = ws_id AND p.key = 'DEMO';
  IF proj_id IS NULL THEN
    INSERT INTO projects (workspace_id, name, key, description, next_issue_number)
    VALUES (ws_id, 'Produto Demo', 'DEMO', 'Projeto de demonstração do 1984.', 12)
    RETURNING id INTO proj_id;
  END IF;
  INSERT INTO project_members (project_id, user_id) VALUES (proj_id, usr_id) ON CONFLICT DO NOTHING;

  SELECT s.id INTO spr_id FROM sprints s WHERE s.project_id = proj_id AND s.name = 'Sprint atual';
  IF spr_id IS NULL THEN
    INSERT INTO sprints (project_id, name, start_date, end_date, started_at)
    VALUES (proj_id, 'Sprint atual', current_date - 7, current_date + 7, now())
    RETURNING id INTO spr_id;
  END IF;

  SELECT s.id INTO old_spr_id FROM sprints s WHERE s.project_id = proj_id AND s.name = 'Sprint anterior';
  IF old_spr_id IS NULL THEN
    INSERT INTO sprints (project_id, name, start_date, end_date, started_at)
    VALUES (proj_id, 'Sprint anterior', current_date - 28, current_date - 14, now() - interval '15 days')
    RETURNING id INTO old_spr_id;
  END IF;

  INSERT INTO releases (project_id, version, name, description, status, target_date)
  VALUES (proj_id, 'v0.1.0-demo', 'Primeira entrega', 'Release criada para inspeção visual.', 'active', current_date + 14)
  ON CONFLICT (project_id, version) DO UPDATE SET status = EXCLUDED.status, target_date = EXCLUDED.target_date
  RETURNING id INTO rel_id;

  INSERT INTO issues (project_id, number, title, description, type, status, priority, assignee_id, reporter_id, estimated_minutes, sprint_id, release_id, closed_at)
  VALUES
    (proj_id, 1, 'Finalizar tela de autenticação', 'Revisar estados de erro e sucesso.', 'feature', 'done', 'high', usr_id, usr_id, 120, spr_id, rel_id, now()),
    (proj_id, 2, 'Adicionar progresso da sprint', 'Exibir progresso no cabeçalho da sprint.', 'improvement', 'done', 'medium', usr_id, usr_id, 90, spr_id, rel_id, now()),
    (proj_id, 3, 'Revisar quadro de tarefas', 'Conferir arrastar e soltar entre colunas.', 'task', 'done', 'medium', usr_id, usr_id, 60, spr_id, rel_id, now()),
    (proj_id, 4, 'Ajustar espaçamento mobile', 'Validar layout em telas pequenas.', 'improvement', 'done', 'low', usr_id, usr_id, 45, spr_id, rel_id, now()),
    (proj_id, 5, 'Criar página de onboarding', 'Preparar primeiro acesso do usuário.', 'feature', 'in_progress', 'high', usr_id, usr_id, 180, spr_id, rel_id, NULL),
    (proj_id, 6, 'Adicionar filtros por responsável', 'Permitir filtrar tarefas por pessoa.', 'feature', 'in_progress', 'medium', usr_id, usr_id, 120, spr_id, rel_id, NULL),
    (proj_id, 7, 'Documentar atalhos', 'Listar atalhos úteis na ajuda.', 'task', 'todo', 'low', usr_id, usr_id, 30, spr_id, rel_id, NULL),
    (proj_id, 8, 'Validar acessibilidade do quadro', 'Revisar foco e navegação por teclado.', 'task', 'review', 'medium', usr_id, usr_id, 90, spr_id, rel_id, NULL),
    (proj_id, 9, 'Mapear fluxo de convites', 'Registrar decisões e próximos passos.', 'task', 'done', 'medium', usr_id, usr_id, 60, old_spr_id, NULL, now()),
    (proj_id, 10, 'Melhorar busca global', 'Suportar referência e título.', 'improvement', 'done', 'low', usr_id, usr_id, 90, old_spr_id, NULL, now()),
    (proj_id, 11, 'Importar tarefas antigas', 'Backlog inicial do produto.', 'task', 'backlog', 'low', usr_id, usr_id, 60, NULL, NULL, NULL)
  ON CONFLICT (project_id, number) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    type = EXCLUDED.type,
    status = EXCLUDED.status,
    priority = EXCLUDED.priority,
    assignee_id = EXCLUDED.assignee_id,
    sprint_id = EXCLUDED.sprint_id,
    release_id = EXCLUDED.release_id,
    closed_at = EXCLUDED.closed_at,
    updated_at = now();

  INSERT INTO comments (issue_id, user_id, body)
  SELECT i.id, usr_id, 'Ambiente demo pronto para revisão visual.'
  FROM issues i
  WHERE i.project_id = proj_id AND i.number = 5
    AND NOT EXISTS (SELECT 1 FROM comments c WHERE c.issue_id = i.id AND c.body = 'Ambiente demo pronto para revisão visual.');

  INSERT INTO time_entries (issue_id, user_id, started_at, ended_at, duration_seconds, description)
  SELECT i.id, usr_id, now() - interval '2 hours', now() - interval '1 hour 15 minutes', 2700, 'Revisão visual demo'
  FROM issues i
  WHERE i.project_id = proj_id AND i.number = 2
    AND NOT EXISTS (SELECT 1 FROM time_entries t WHERE t.issue_id = i.id AND t.description = 'Revisão visual demo');
END $$;
