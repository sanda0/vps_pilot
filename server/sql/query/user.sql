-- name: CreateUser :one
INSERT INTO users (username, email, password_hash)
VALUES ($1, $2, $3)
RETURNING *;

-- name: FindUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: FindUserById :one
SELECT * FROM users WHERE id = $1;