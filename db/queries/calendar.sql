-- name: CalendarRange :many
SELECT i.id, i.number, i.title, p.key AS project_key, i.start_date, i.due_date,
       NULL::date AS target_date, 'issue'::text AS item_kind
FROM issues i JOIN projects p ON p.id = i.project_id
WHERE p.workspace_id = sqlc.arg(workspace_id)
 AND (sqlc.arg(project_key)::text = '' OR p.key = sqlc.arg(project_key))
 AND ((i.start_date >= sqlc.arg(date_from)::date AND i.start_date < sqlc.arg(date_to)::date)
   OR (i.due_date >= sqlc.arg(date_from)::date AND i.due_date < sqlc.arg(date_to)::date))
UNION ALL
SELECT r.id, 0, coalesce(nullif(r.name, ''), r.version), p.key, NULL::date, NULL::date,
       r.target_date, 'release'::text
FROM releases r JOIN projects p ON p.id = r.project_id
WHERE p.workspace_id = sqlc.arg(workspace_id)
 AND (sqlc.arg(project_key)::text = '' OR p.key = sqlc.arg(project_key))
 AND r.target_date >= sqlc.arg(date_from)::date AND r.target_date < sqlc.arg(date_to)::date
ORDER BY target_date NULLS LAST, due_date NULLS LAST, start_date NULLS LAST
LIMIT 1000;
