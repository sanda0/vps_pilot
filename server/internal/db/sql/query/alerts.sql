-- name: CreateAlert :one
INSERT INTO alerts(
    node_id,
    metric,
    duration,
    threshold,
    net_rece_threshold,
    net_send_threshold,
    email,
    discord_webhook,
    slack_webhook,
    is_active
  )
values (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10
  )
RETURNING *;

-- name: GetAlerts :many
SELECT * FROM alerts
WHERE node_id = $1
LIMIT $2 OFFSET $3;

-- name: GetAlert :one
SELECT * FROM alerts
WHERE id = $1;

-- name: UpdateAlert :one
UPDATE alerts
SET node_id = $2,
  metric = $3,
  duration = $4,
  threshold = $5,
  net_rece_threshold = $6,
  net_send_threshold = $7,
  email = $8,
  discord_webhook = $9,
  slack_webhook = $10,
  is_active = $11
WHERE id = $1
RETURNING *;

-- name: DeleteAlert :exec
DELETE FROM alerts
WHERE id = $1;

-- name: ActivateAlert :exec
UPDATE alerts
SET is_active = true
WHERE id = $1;

-- name: DeactivateAlert :exec
UPDATE alerts
SET is_active = false
WHERE id = $1;

