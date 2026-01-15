-- name: CreateProject :one
INSERT INTO projects (name, description, node_id, repo_url, branch, deploy_path, status)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetProject :one
SELECT * FROM projects WHERE id = ?;

-- name: ListProjects :many
SELECT * FROM projects 
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListProjectsByNode :many
SELECT * FROM projects 
WHERE node_id = ? 
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: UpdateProject :one
UPDATE projects 
SET name = ?,
    description = ?,
    repo_url = ?,
    branch = ?,
    deploy_path = ?,
    status = ?,
    updated_at = strftime('%s', 'now')
WHERE id = ?
RETURNING *;

-- name: UpdateProjectStatus :one
UPDATE projects 
SET status = ?,
    updated_at = strftime('%s', 'now')
WHERE id = ?
RETURNING *;

-- name: UpdateProjectLastDeployed :one
UPDATE projects 
SET last_deployed_at = strftime('%s', 'now'),
    updated_at = strftime('%s', 'now')
WHERE id = ?
RETURNING *;

-- name: DeleteProject :execrows
DELETE FROM projects WHERE id = ?;

-- name: GetProjectWithNode :one
SELECT 
    p.*,
    n.name as node_name,
    n.ip as node_ip
FROM projects p
LEFT JOIN nodes n ON p.node_id = n.id
WHERE p.id = ?;

-- name: ListProjectsWithNodes :many
SELECT 
    p.*,
    n.name as node_name,
    n.ip as node_ip
FROM projects p
LEFT JOIN nodes n ON p.node_id = n.id
ORDER BY p.created_at DESC
LIMIT ? OFFSET ?;

-- name: CountProjects :one
SELECT COUNT(*) FROM projects;

-- name: CountProjectsByNode :one
SELECT COUNT(*) FROM projects WHERE node_id = ?;
