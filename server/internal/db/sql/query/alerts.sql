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
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
  )
RETURNING *;

-- name: GetAlerts :many
SELECT * FROM alerts
WHERE node_id = ?
ORDER BY id DESC
LIMIT ? OFFSET ?;

-- name: GetAlert :one
SELECT * FROM alerts
WHERE id = ?;

-- name: UpdateAlert :one
UPDATE alerts
SET node_id = ?,
  metric = ?,
  duration = ?,
  threshold = ?,
  net_rece_threshold = ?,
  net_send_threshold = ?,
  email = ?,
  discord_webhook = ?,
  slack_webhook = ?,
  is_active = ?
WHERE id = ?
RETURNING *;

-- name: DeleteAlert :exec
DELETE FROM alerts
WHERE id = ?;

-- name: ActivateAlert :exec
UPDATE alerts
SET is_active = 1
WHERE id = ?;

-- name: DeactivateAlert :exec
UPDATE alerts
SET is_active = 0
WHERE id = ?;

-- name: GetActiveAlertsByNodeAndMetric :many
SELECT a.*,n.name as node_name,n.ip as node_ip FROM alerts a
join nodes n on a.node_id = n.id
WHERE node_id = ? AND metric = ? AND is_active = 1;
