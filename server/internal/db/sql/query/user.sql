-- name: CreateUser :one
INSERT INTO users (username, email, password_hash)
VALUES (?, ?, ?)
RETURNING *;

-- name: FindUserByEmail :one
SELECT * FROM users WHERE email = ?;

-- name: FindUserById :one
SELECT * FROM users WHERE id = ?;
-- name: SaveGitHubToken :exec
UPDATE users 
SET github_token = ?, updated_at = ? 
WHERE id = ?;

-- name: GetGitHubToken :one
SELECT github_token FROM users WHERE id = ?;

-- name: RemoveGitHubToken :exec
UPDATE users 
SET github_token = NULL, updated_at = ? 
WHERE id = ?;
