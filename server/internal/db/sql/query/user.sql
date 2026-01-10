-- name: CreateUser :one
INSERT INTO users (username, email, password_hash)
VALUES (?, ?, ?)
RETURNING *;

-- name: FindUserByEmail :one
SELECT * FROM users WHERE email = ?;

-- name: FindUserById :one
SELECT * FROM users WHERE id = ?;