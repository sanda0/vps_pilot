-- name: CreateJoinToken :one
INSERT INTO agent_join_tokens (token_hash, node_id, created_by, expires_at)
VALUES (?, ?, ?, ?)
RETURNING *;
-- name: ConsumeJoinToken :one
-- Marks an unused, unexpired token as used. No row = invalid, used or expired.
UPDATE agent_join_tokens
SET used_at = strftime('%s', 'now')
WHERE token_hash = ?
  AND used_at IS NULL
  AND expires_at > strftime('%s', 'now')
RETURNING *;
-- name: SetJoinTokenUsedBy :exec
UPDATE agent_join_tokens
SET used_by_node_id = ?
WHERE id = ?;
-- name: DeleteExpiredJoinTokens :execrows
DELETE FROM agent_join_tokens
WHERE used_at IS NULL
  AND expires_at <= strftime('%s', 'now');
