-- name: UpsertProject :one
INSERT INTO projects (node_id, name, path, tech, commands, logs, backups, discovered_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, strftime('%s', 'now'), strftime('%s', 'now'))
ON CONFLICT (node_id, path) DO UPDATE SET
    name       = excluded.name,
    tech       = excluded.tech,
    commands   = excluded.commands,
    logs       = excluded.logs,
    backups    = excluded.backups,
    updated_at = strftime('%s', 'now')
RETURNING id, node_id, name, path, tech, commands, logs, backups, discovered_at, updated_at;

-- name: GetProject :one
SELECT id, node_id, name, path, tech, commands, logs, backups, discovered_at, updated_at
FROM projects
WHERE id = ?;

-- name: GetProjectWithNode :one
SELECT
    p.id, p.node_id, p.name, p.path, p.tech, p.commands, p.logs, p.backups, p.discovered_at, p.updated_at,
    n.name AS node_name,
    n.ip   AS node_ip
FROM projects p
LEFT JOIN nodes n ON p.node_id = n.id
WHERE p.id = ?;

-- name: ListProjects :many
SELECT id, node_id, name, path, tech, commands, logs, backups, discovered_at, updated_at
FROM projects
ORDER BY updated_at DESC
LIMIT ? OFFSET ?;

-- name: ListProjectsByNode :many
SELECT id, node_id, name, path, tech, commands, logs, backups, discovered_at, updated_at
FROM projects
WHERE node_id = ?
ORDER BY updated_at DESC
LIMIT ? OFFSET ?;

-- name: ListProjectsWithNodes :many
SELECT
    p.id, p.node_id, p.name, p.path, p.tech, p.commands, p.logs, p.backups, p.discovered_at, p.updated_at,
    n.name AS node_name,
    n.ip   AS node_ip
FROM projects p
LEFT JOIN nodes n ON p.node_id = n.id
ORDER BY p.updated_at DESC
LIMIT ? OFFSET ?;

-- name: DeleteProject :execrows
DELETE FROM projects WHERE id = ?;

-- name: DeleteProjectsByNode :execrows
DELETE FROM projects WHERE node_id = ?;

-- name: CountProjects :one
SELECT COUNT(*) FROM projects;

-- name: CountProjectsByNode :one
SELECT COUNT(*) FROM projects WHERE node_id = ?;
